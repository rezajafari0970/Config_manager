package panel

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/rezajafari0970/Config_manager/internal/residential"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func adsPolicy(t *testing.T, s *Store) {
	p := resPolicy(t, s)
	p.Repeats = 3
	b, _ := json.Marshal(p)
	if _, e := s.db.Exec("UPDATE residential_settings SET policy=?,policy_hash=?", string(b), p.Digest()); e != nil {
		t.Fatal(e)
	}
}
func adsCompletion(j residential.Job, same bool) residential.Completion {
	c := resCompletion(j)
	c.Report.IPFacts = nil
	for i := range c.Report.Controls {
		o := &c.Report.Controls[i]
		o.Family = 4
		o.Transport = "tcp"
		o.Path = "single-proxy-route"
		o.IP = "1.1.1.1"
		o.SameConnection = true
		o.Connections = 1
		o.ActualDomainRequest = true
		o.ConnectionID = fmt.Sprintf("control-%d-unique-connection", i)
		o.ResponseSHA256 = strings.Repeat("c", 64)
	}
	for i := range c.Report.Observations {
		o := &c.Report.Observations[i]
		if same {
			o.IP = "1.1.1.1"
		}
		o.Nonce = fmt.Sprintf("ads-%d-unique-nonce-12345", i)
		o.Attestation = base64.StdEncoding.EncodeToString(ed25519.Sign(ed25519.NewKeyFromSeed([]byte(strings.Repeat("s", 32))), residential.SigningPayload(*o)))
	}
	return c
}
func seedAds(t *testing.T, s *Store, n int) (string, *residential.Job) {
	t.Helper()
	source, _ := addConversion(t, s, strings.Replace(conversionSample, "192.0.2.10", fmt.Sprintf("192.0.2.%d", n), 1))
	j, e := s.ClaimResidential(context.Background())
	if e != nil || j == nil {
		t.Fatal(e)
	}
	if e = s.CompleteResidential(context.Background(), adsCompletion(*j, false)); e != nil {
		t.Fatal(e)
	}
	return source, j
}
func TestDifferentAdsPoolIsIndependentAndAgeExempt(t *testing.T) {
	s := testStore(t)
	adsPolicy(t, s)
	source, j := seedAds(t, s, 11)
	ctx := context.Background()
	for _, f := range []string{"configs.jsonl", "next.json", "raw.txt"} {
		if w := resRequest(s, "ads:all", f); w.Code != 200 || !strings.Contains(w.Header().Get("X-Evidence-Class"), "not-residential") {
			t.Fatal(f, w.Code, w.Body.String())
		}
		if w := resRequest(s, "all", f); w.Code != 503 {
			t.Fatal("route difference became Residential", f, w.Code)
		}
	}
	s.db.Exec("UPDATE artifacts SET expires=1;UPDATE residential_records SET retain_until=1,first_seen=first_seen-108000000")
	if e := s.Expire(ctx, time.Now()); e != nil {
		t.Fatal(e)
	}
	if e := s.Delete(ctx, source, 1); e != nil {
		t.Fatal(e)
	}
	if e := s.vault.Sweep(ctx, s.Referenced); e != nil {
		t.Fatal(e)
	}
	if w := resRequest(s, "ads:all", "raw.txt"); w.Code != 200 {
		t.Fatal("source/TTL/GC lost accepted route", w.Code)
	}
	s.db.Exec("UPDATE residential_records SET valid_until=0,next_run=0 WHERE hash=?", j.Hash)
	if w := resRequest(s, "ads:all", "raw.txt"); w.Code != 503 {
		t.Fatal("expired proof published")
	}
	fresh, e := s.ClaimResidential(ctx)
	if e != nil || fresh == nil {
		t.Fatal("retained not retested", e)
	}
	if e = s.CompleteResidential(ctx, adsCompletion(*fresh, false)); e != nil {
		t.Fatal(e)
	}
	if w := resRequest(s, "ads:all", "raw.txt"); w.Code != 200 {
		t.Fatal("fresh proof not restored")
	}
}
func TestDifferentAdsConfirmedLossRevokesAndFencesStaleJobs(t *testing.T) {
	s := testStore(t)
	adsPolicy(t, s)
	_, j := seedAds(t, s, 12)
	ctx := context.Background()
	for attempt := 0; attempt < 3; attempt++ {
		if _, e := s.db.Exec("UPDATE residential_records SET next_run=0,negative_worker='',negative_last=CASE WHEN negative_last>0 THEN ? ELSE 0 END,negative_first=CASE WHEN negative_first>0 THEN ? ELSE 0 END WHERE hash=?", time.Now().Add(-40*time.Second).UnixMilli(), time.Now().Add(-100*time.Second).UnixMilli(), j.Hash); e != nil {
			t.Fatal(e)
		}
		fresh, e := s.ClaimResidential(ctx)
		if e != nil || fresh == nil {
			t.Fatal(e)
		}
		if e = s.CompleteResidential(ctx, adsCompletion(*fresh, true)); e != nil {
			t.Fatal(e)
		}
		if w := resRequest(s, "ads:all", "raw.txt"); w.Code != 503 {
			t.Fatal("loss still published")
		}
		var state string
		s.db.QueryRow("SELECT state FROM residential_records WHERE hash=?", j.Hash).Scan(&state)
		if attempt < 2 && state != "QUARANTINED" || attempt == 2 && state != "DELETED" {
			t.Fatal(attempt, state)
		}
	}
	if e := s.CompleteResidential(ctx, adsCompletion(*j, false)); e != ErrConflict {
		t.Fatal("stale work restored deleted member", e)
	}
	if used, e := s.Referenced(ctx, j.Hash); e != nil || used {
		t.Fatal("revoked original referenced", used, e)
	}
}

func TestDifferentAdsOutputsParityConcurrentRotationAndReopen(t *testing.T) {
	s := testStore(t)
	adsPolicy(t, s)
	source, first := seedAds(t, s, 21)
	_, second := seedAds(t, s, 22)
	all := resRequest(s, "ads:all", "configs.jsonl")
	raw := resRequest(s, "ads:all", "raw.txt")
	if all.Code != 200 || raw.Code != 200 || all.Header().Get("X-Residential-Generation") != raw.Header().Get("X-Residential-Generation") {
		t.Fatal("generation parity")
	}
	want := map[string]bool{}
	for _, line := range bytes.Split(bytes.TrimSpace(all.Body.Bytes()), []byte{'\n'}) {
		if !json.Valid(line) {
			t.Fatal("invalid JSONL")
		}
		want[residential.Digest(line)] = true
	}
	if len(want) != 2 || !bytes.Contains(raw.Body.Bytes(), first.Raw) || !bytes.Contains(raw.Body.Bytes(), second.Raw) {
		t.Fatal("raw/template membership mismatch")
	}
	other, e := Open(filepath.Join(filepath.Dir(s.vault.Directory()), "panel.db"), s.key)
	if e != nil {
		t.Fatal(e)
	}
	defer other.Close()
	var wg sync.WaitGroup
	results := make(chan map[string]string, 2)
	for _, store := range []*Store{s, other} {
		wg.Add(1)
		go func(store *Store) {
			defer wg.Done()
			w := resRequest(store, "ads:all", "next.json")
			results <- map[string]string{"status": fmt.Sprint(w.Code), "hash": w.Header().Get("X-Content-SHA256"), "round": w.Header().Get("X-Export-Cycle"), "generation": w.Header().Get("X-Residential-Generation")}
		}(store)
	}
	wg.Wait()
	close(results)
	seen := map[string]bool{}
	round := ""
	for result := range results {
		if result["status"] != "200" || !want[result["hash"]] || seen[result["hash"]] || result["generation"] != all.Header().Get("X-Residential-Generation") {
			t.Fatal(result)
		}
		seen[result["hash"]] = true
		if round != "" && round != result["round"] {
			t.Fatal("premature round")
		}
		round = result["round"]
	}
	scoped := resRequest(s, "ads:"+source, "next.json")
	if scoped.Code != 200 || scoped.Header().Get("X-Original-SHA256") != first.Hash {
		t.Fatal("per-link rotation/membership")
	}
	next := resRequest(other, "ads:all", "next.json")
	if next.Code != 200 || next.Header().Get("X-Export-Cycle") == round {
		t.Fatal("durable next round")
	}
}

func TestDifferentAdsUncertainRetestQuarantinesWithoutDeletion(t *testing.T) {
	for _, mode := range []string{"rotation", "outage", "missing-attribution", "unbound-same-route"} {
		t.Run(mode, func(t *testing.T) {
			s := testStore(t)
			adsPolicy(t, s)
			_, job := seedAds(t, s, 23)
			s.db.Exec("UPDATE residential_records SET next_run=0,negative_count=2,negative_first=?,negative_last=? WHERE hash=?", time.Now().UnixMilli()-100000, time.Now().UnixMilli()-40000, job.Hash)
			j, e := s.ClaimResidential(context.Background())
			if e != nil || j == nil {
				t.Fatal(e)
			}
			c := adsCompletion(*j, false)
			switch mode {
			case "unbound-same-route":
				c = adsCompletion(*j, true)
				c.Candidate = nil
				c.Report.Variants[0].Hash = strings.Repeat("f", 64)
			case "rotation":
				for i := range c.Report.Controls {
					if c.Report.Controls[i].Wave == 2 {
						c.Report.Controls[i].IP = "9.9.9.9"
					}
				}
			case "outage":
				c.Report.InfrastructureOK = false
			case "missing-attribution":
				for i := range c.Report.Observations {
					c.Report.Observations[i].Attestation = ""
				}
			}
			if e = s.CompleteResidential(context.Background(), c); e != nil {
				t.Fatal(e)
			}
			var state string
			s.db.QueryRow("SELECT state FROM residential_records WHERE hash=?", job.Hash).Scan(&state)
			if state != "QUARANTINED" {
				t.Fatal(state)
			}
			for _, format := range []string{"configs.jsonl", "next.json", "raw.txt"} {
				if resRequest(s, "ads:all", format).Code != 503 {
					t.Fatal("uncertainty published")
				}
			}
			if used, _ := s.Referenced(context.Background(), job.Hash); !used {
				t.Fatal("uncertainty deleted raw")
			}
		})
	}
}

func TestDifferentAdsSharedCircuitLatchesBeforeDestructiveThreshold(t *testing.T) {
	s := testStore(t)
	adsPolicy(t, s)
	_, a := seedAds(t, s, 31)
	_, b := seedAds(t, s, 32)
	ctx := context.Background()
	now := time.Now().UnixMilli()
	s.db.Exec("UPDATE residential_records SET negative_count=1,negative_first=?,negative_last=? WHERE hash=?", now-60000, now-30000, b.Hash)
	s.db.Exec("UPDATE residential_records SET next_run=0 WHERE hash=?", a.Hash)
	fresh, e := s.ClaimResidential(ctx)
	if e != nil || fresh == nil || fresh.Hash != a.Hash {
		t.Fatal(e)
	}
	if e = s.CompleteResidential(ctx, adsCompletion(*fresh, true)); e != nil {
		t.Fatal(e)
	}
	var circuit int64
	s.db.QueryRow("SELECT circuit_until FROM residential_settings").Scan(&circuit)
	if circuit <= now {
		t.Fatal("shared failure not latched at first vote")
	}
	// The other worker stalls. Its row falls outside the recent window, but the
	// existing cooldown must still fence the first record's final fresh vote.
	s.db.Exec("UPDATE residential_records SET negative_last=? WHERE hash=?", now-130000, b.Hash)
	s.db.Exec("UPDATE residential_records SET next_run=0,negative_worker='',negative_count=2,negative_first=?,negative_last=? WHERE hash=?", now-100000, now-40000, a.Hash)
	fresh, e = s.ClaimResidential(ctx)
	if e != nil || fresh == nil || fresh.Hash != a.Hash {
		t.Fatal(e)
	}
	if e = s.CompleteResidential(ctx, adsCompletion(*fresh, true)); e != nil {
		t.Fatal(e)
	}
	var state string
	s.db.QueryRow("SELECT state FROM residential_records WHERE hash=?", a.Hash).Scan(&state)
	if state != "QUARANTINED" {
		t.Fatal("latched outage allowed deletion", state)
	}
}
