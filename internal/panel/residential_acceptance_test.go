package panel

import (
	"context"
	"errors"
	"github.com/rezajafari0970/Config_manager/internal/residential"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResidentialFreshNegativeConfirmationAndSharedCircuit(t *testing.T) {
	for _, mode := range []string{"datacenter", "connection", "shared"} {
		t.Run(mode, func(t *testing.T) {
			s := testStore(t)
			resPolicy(t, s)
			_, job, _ := resSeed(t, s, 1)
			ctx := context.Background()
			if mode == "shared" {
				_, other, _ := resSeed(t, s, 2)
				s.db.Exec("UPDATE residential_records SET negative_count=1,negative_last=? WHERE hash=?", time.Now().UnixMilli(), other.Hash)
			}
			// Simulate two earlier independently recorded, spaced failures; the third
			// is a newly claimed assessment. No clock sleeps or forged future evidence.
			now := time.Now().UnixMilli()
			s.db.Exec("UPDATE residential_records SET next_run=0,negative_count=2,negative_first=?,negative_last=? WHERE hash=?", now-100000, now-40000, job.Hash)
			j, e := s.ClaimResidential(ctx)
			if e != nil || j == nil || j.Hash != job.Hash {
				t.Fatal(j, e)
			}
			completion := resCompletion(*j)
			if mode == "connection" {
				for i := range completion.Report.Variants {
					completion.Report.Variants[i].Health = "UNHEALTHY"
				}
				for i := range completion.Report.Controls {
					completion.Report.Controls[i].Outcome = "failed"
					completion.Report.Controls[i].TLSVerified = false
				}
				for _, host := range []string{"www.cloudflare.com", "api.ipify.org"} {
					completion.Report.InfrastructureControls = append(completion.Report.InfrastructureControls, residential.Observation{Scope: residential.Scope{Domain: host}, At: time.Now().UnixMilli(), Outcome: "passed", TLSVerified: true, OriginVerified: true})
				}
			} else {
				for i := range completion.Report.IPFacts {
					completion.Report.IPFacts[i].Class = "Datacenter"
				}
			}
			for i := range completion.Report.IPFacts {
				completion.Report.IPFacts[i].MAC = residential.IPFactMAC(completion.Report.IPFacts[i], s.key)
			}
			if e = s.CompleteResidential(ctx, completion); e != nil {
				t.Fatal(e)
			}
			var state string
			s.db.QueryRow("SELECT state FROM residential_records WHERE hash=?", job.Hash).Scan(&state)
			want := "DELETED"
			if mode == "shared" {
				want = "QUARANTINED"
			}
			if state != want {
				t.Fatalf("%s expected %s", state, want)
			}
			for _, format := range []string{"configs.jsonl", "next.json", "raw.zip"} {
				w := resRequest(s, "all", format)
				if mode != "shared" && w.Code != 503 {
					t.Fatal(format, w.Code)
				}
			}
		})
	}
}

func TestResidentialLegacyOutputPreparedBeforeRevocationIsBlocked(t *testing.T) {
	for _, mode := range []string{"revoke", "expire", "not-modified"} {
		t.Run(mode, func(t *testing.T) {
			s := testStore(t)
			resPolicy(t, s)
			_, job, _ := resSeed(t, s, 1)
			server, _ := NewServer(s, "http://127.0.0.1:18191")
			request := httptest.NewRequest("GET", "http://127.0.0.1:18191/output/archive.zip", nil)
			w := httptest.NewRecorder()
			server.guardLegacyPublication(w, request, http.HandlerFunc(func(out http.ResponseWriter, r *http.Request) {
				if mode == "not-modified" {
					out.WriteHeader(304)
				} else {
					out.Write([]byte("sensitive-prepared-configuration"))
				}
				if mode == "expire" {
					s.db.Exec("UPDATE residential_records SET valid_until=0")
					return
				}
				tx, e := s.db.BeginTx(r.Context(), nil)
				if e != nil {
					t.Fatal(e)
				}
				if e = s.revokeResidential(r.Context(), tx, job.Hash, "manual_delete", true, time.Now().UnixMilli()); e != nil {
					tx.Rollback()
					t.Fatal(e)
				}
				if e = tx.Commit(); e != nil {
					t.Fatal(e)
				}
			}))
			if w.Code != 503 || w.Body.String() == "sensitive-prepared-configuration" {
				t.Fatal("stale prepared bytes served", w.Code)
			}
		})
	}
}

func TestResidentialJournalCrashReplayTruncationAndRestart(t *testing.T) {
	s := testStore(t)
	resPolicy(t, s)
	_, job, _ := resSeed(t, s, 1)
	ctx := context.Background()
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.revokeResidential(ctx, tx, job.Hash, "manual_delete", true, time.Now().UnixMilli()); e != nil {
		t.Fatal(e)
	}
	tx.Rollback() // crash after durable journal, before DB commit
	if e = s.residentialBarrier(ctx); !errors.Is(e, errResidentialJournal) {
		t.Fatal("uncommitted revocation not fenced", e)
	}
	path := filepath.Join(filepath.Dir(s.vault.Directory()), "panel.db")
	if e = s.initResidentialJournal(ctx, path); e != nil {
		t.Fatal(e)
	}
	if w := resRequest(s, "all", "next.json"); w.Code != 503 {
		t.Fatal("crashed revocation resurrected")
	}
	if e = os.Truncate(s.residentialJournal, 0); e != nil {
		t.Fatal(e)
	}
	if e = s.initResidentialJournal(ctx, path); e == nil {
		t.Fatal("truncated journal accepted")
	}
}

func TestResidentialPolicyChangeDisableAndStartupFence(t *testing.T) {
	for _, mode := range []string{"policy", "disable", "restart"} {
		t.Run(mode, func(t *testing.T) {
			s := testStore(t)
			p := resPolicy(t, s)
			_, job, _ := resSeed(t, s, 1)
			ctx := context.Background()
			switch mode {
			case "policy":
				_, trust, e := residentialSettings(ctx, s.db)
				if e != nil {
					t.Fatal(e)
				}
				p.Domains = append(p.Domains, "pubads.g.doubleclick.net")
				if e = s.ConfigureResidential(ctx, p, trust); e != nil {
					t.Fatal(e)
				}
			case "disable":
				var id string
				s.db.QueryRow("SELECT identity FROM residential_owners WHERE hash=?", job.Hash).Scan(&id)
				if _, e := s.ManageConfigs(ctx, ConfigBatch{Action: "disable", IDs: []string{id}}); e != nil {
					t.Fatal(e)
				}
			case "restart":
				if e := s.PrepareResidentialServing(ctx); e != nil {
					t.Fatal(e)
				}
			}
			if w := resRequest(s, "all", "configs.jsonl"); w.Code != 503 {
				t.Fatal("stale proof available", w.Code)
			}
			if used, _ := s.Referenced(ctx, job.Hash); !used {
				t.Fatal("invalidation deleted original")
			}
		})
	}
}

func TestResidentialAutomaticReadmissionRequiresNewEvidence(t *testing.T) {
	s := testStore(t)
	resPolicy(t, s)
	_, job, _ := resSeed(t, s, 1)
	ctx := context.Background()
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.revokeResidential(ctx, tx, job.Hash, "verified_nonresidential_ads_sample", false, time.Now().UnixMilli()-1); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
	source := testSource()
	source.URL = "https://example.com/fresh-readmission"
	id, e := s.Save(ctx, "", 0, source)
	if e != nil {
		t.Fatal(e)
	}
	fetch := claimOne(t, s, id)
	if e = s.Complete(ctx, fetch, fixtureResult(t, s, string(job.Raw)), time.Now()); e != nil {
		t.Fatal(e)
	}
	if w := resRequest(s, "all", "configs.jsonl"); w.Code != 503 {
		t.Fatal("fetch alone republished")
	}
	fresh, e := s.ClaimResidential(ctx)
	if e != nil || fresh == nil || fresh.Epoch <= job.Epoch {
		t.Fatal("fresh epoch", fresh, e)
	}
	completion := resCompletion(*fresh)
	completion.Report.Started = completion.Report.Finished
	for i := range completion.Report.IPFacts {
		completion.Report.IPFacts[i].MAC = residential.IPFactMAC(completion.Report.IPFacts[i], s.key)
	}
	if e = s.CompleteResidential(ctx, completion); e != nil {
		t.Fatal(e)
	}
	if w := resRequest(s, "all", "configs.jsonl"); w.Code != 200 {
		t.Fatal("new valid evidence not admitted", w.Code, w.Body.String())
	}
	var blocked int
	if e = s.db.QueryRow("SELECT count(*) FROM residential_blocked_identities WHERE identity IN(SELECT identity FROM residential_owners WHERE hash=?)", job.Hash).Scan(&blocked); e != nil || blocked != 0 {
		t.Fatal("identity remained blocked", blocked, e)
	}
}

func TestResidentialSourceRoundsIndependentAndDurable(t *testing.T) {
	s := testStore(t)
	resPolicy(t, s)
	source, job, _ := resSeed(t, s, 1)
	resSeed(t, s, 2)
	first := resRequest(s, "all", "next.json")
	scoped := resRequest(s, source, "next.json")
	if first.Code != 200 || scoped.Code != 200 || scoped.Header().Get("X-Original-SHA256") != job.Hash {
		t.Fatal("scope membership")
	}
	other, e := Open(filepath.Join(filepath.Dir(s.vault.Directory()), "panel.db"), s.key)
	if e != nil {
		t.Fatal(e)
	}
	defer other.Close()
	second := resRequest(other, "all", "next.json")
	if second.Code != 200 || first.Header().Get("X-Export-Cycle") != second.Header().Get("X-Export-Cycle") || first.Header().Get("X-Original-SHA256") == second.Header().Get("X-Original-SHA256") {
		t.Fatal("rotation lost at reopen")
	}
}

func TestResidentialForgedFactsUnboundNegativeAndManualAlias(t *testing.T) {
	for _, mode := range []string{"unsigned-intel", "wrong-target", "manual-alias"} {
		t.Run(mode, func(t *testing.T) {
			s := testStore(t)
			resPolicy(t, s)
			_, job, _ := resSeed(t, s, 1)
			ctx := context.Background()
			if mode == "manual-alias" {
				tx, _ := s.db.BeginTx(ctx, nil)
				if e := s.revokeResidential(ctx, tx, job.Hash, "manual_delete", true, time.Now().UnixMilli()); e != nil {
					t.Fatal(e)
				}
				if e := tx.Commit(); e != nil {
					t.Fatal(e)
				}
				source := testSource()
				source.URL = "https://example.com/manual-alias"
				id, e := s.Save(ctx, "", 0, source)
				if e != nil {
					t.Fatal(e)
				}
				fetch := claimOne(t, s, id)
				if e = s.Complete(ctx, fetch, fixtureResult(t, s, string(job.Raw)+"-new-label"), time.Now()); e != nil {
					t.Fatal(e)
				}
				var n int
				s.db.QueryRow("SELECT count(*) FROM memberships WHERE source_id=?", id).Scan(&n)
				if n != 0 {
					t.Fatal("manual alias reingested", n)
				}
			} else {
				s.db.Exec("UPDATE residential_records SET next_run=0,negative_count=2,negative_first=?,negative_last=? WHERE hash=?", time.Now().UnixMilli()-100000, time.Now().UnixMilli()-40000, job.Hash)
				j, e := s.ClaimResidential(ctx)
				if e != nil || j == nil {
					t.Fatal(e)
				}
				c := resCompletion(*j)
				for i := range c.Report.IPFacts {
					c.Report.IPFacts[i].Class = "Datacenter"
					if mode == "unsigned-intel" {
						c.Report.IPFacts[i].MAC = ""
					} else {
						c.Report.IPFacts[i].MAC = residential.IPFactMAC(c.Report.IPFacts[i], s.key)
					}
				}
				if mode == "wrong-target" {
					c.Candidate = nil
					c.Report.Variants[0].Hash = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
				}
				if e = s.CompleteResidential(ctx, c); e != nil {
					t.Fatal(e)
				}
				var state string
				s.db.QueryRow("SELECT state FROM residential_records WHERE hash=?", job.Hash).Scan(&state)
				if state != "QUARANTINED" {
					t.Fatal("unverified destructive report", state)
				}
			}
			if w := resRequest(s, "all", "configs.jsonl"); w.Code != 503 {
				t.Fatal("unverified republished")
			}
		})
	}
}

func TestResidentialPartialJournalAppendIsRecoverableOnlyBeyondWatermark(t *testing.T) {
	for _, tail := range []string{"{", "{\"sequence\":1,", "{\"sequence\":1,\"hash\":\"partial"} {
		t.Run(tail, func(t *testing.T) {
			s := testStore(t)
			f, e := os.OpenFile(s.residentialJournal, os.O_WRONLY|os.O_APPEND, 0600)
			if e != nil {
				t.Fatal(e)
			}
			f.WriteString(tail)
			f.Sync()
			f.Close()
			if e = s.residentialBarrier(context.Background()); e == nil {
				t.Fatal("partial journal served")
			}
			if e = s.initResidentialJournal(context.Background(), filepath.Join(filepath.Dir(s.vault.Directory()), "panel.db")); e != nil {
				t.Fatal("uncommitted tail not recovered", e)
			}
			if e = s.residentialBarrier(context.Background()); e != nil {
				t.Fatal(e)
			}
		})
	}
}

func TestResidentialUncertainNewAssessmentDoesNotInvalidateUnrelatedOutput(t *testing.T) {
	s := testStore(t)
	resPolicy(t, s)
	resSeed(t, s, 1)
	ctx := context.Background()
	_, normal := addConversion(t, s, conversionSample)
	if e := s.CompleteProbe(ctx, conversionResult(normal, 0, true)); e != nil {
		t.Fatal(e)
	}
	job, e := s.ClaimResidential(ctx)
	if e != nil || job == nil {
		t.Fatal(e)
	}
	result := resCompletion(*job)
	result.Report.InfrastructureOK = false
	server, _ := NewServer(s, "http://127.0.0.1:18191")
	request := httptest.NewRequest("GET", "http://127.0.0.1:18191/output/archive.zip", nil)
	w := httptest.NewRecorder()
	server.guardLegacyPublication(w, request, http.HandlerFunc(func(out http.ResponseWriter, r *http.Request) {
		out.Write([]byte("already-qualified-member"))
		if e = s.CompleteResidential(ctx, result); e != nil {
			t.Fatal(e)
		}
	}))
	if w.Code != 200 || w.Body.String() != "already-qualified-member" {
		t.Fatal("unrelated unknown assessment invalidates output", w.Code)
	}
}
