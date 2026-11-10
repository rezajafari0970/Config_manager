package panel

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rezajafari0970/Config_manager/internal/convert"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"github.com/rezajafari0970/Config_manager/internal/residential"
)

// These synthetic receipts test admission, never classify a real IP/network.
func resPolicy(t *testing.T, s *Store) residential.Policy {
	t.Helper()
	p := residential.DefaultPolicy()
	p.Domains = []string{"googleads.g.doubleclick.net"}
	p.Families = []int{4}
	p.Transports = []string{"tcp"}
	p.Repeats = 2
	trust := residential.EmptyTrust()
	trust.Providers = map[string]residential.Provider{"fixture-specialist": {Group: "fixture-one", Specialist: true}, "fixture-independent": {Group: "fixture-two"}}
	key := ed25519.NewKeyFromSeed([]byte(strings.Repeat("s", 32)))
	trust.Capabilities["fixture-attestation"] = residential.Capability{Method: "server-flow-ed25519-v1", PublicKey: base64.StdEncoding.EncodeToString(key.Public().(ed25519.PublicKey)), Domains: p.Domains, Transports: p.Transports, Expires: time.Now().Add(time.Hour).UnixMilli(), EvidenceSHA256: strings.Repeat("a", 64)}
	pb, _ := json.Marshal(p)
	tb, _ := json.Marshal(trust)
	if _, e := s.db.Exec("UPDATE residential_settings SET policy=?,policy_hash=?,trust=?,trust_hash=? WHERE id=1", string(pb), p.Digest(), string(tb), trust.Digest()); e != nil {
		t.Fatal(e)
	}
	return p
}
func resCompletion(j residential.Job) residential.Completion {
	plan := residential.NewCoreRegistry().Prepare("xray-26.7.28", j.Raw)
	now := time.Now().UnixMilli()
	r := residential.Report{Version: residential.Version, PolicyHash: j.Policy.Digest(), TemplateVersion: convert.Version, CoreSHA256: probe.CoreSHA256, Worker: "fixture-worker", InfrastructureEpoch: "fixture-epoch", Started: now - 1000, Finished: now, InfrastructureOK: true}
	for _, v := range plan.Variants {
		r.Variants = append(r.Variants, residential.Variant{Name: v.Name, Hash: v.SHA256, RuntimeHash: v.SHA256, Core: "passed", Health: "HEALTHY", Paths: v.Paths, PathsComplete: v.PathsComplete})
		for wave := 1; wave <= j.Policy.Repeats; wave++ {
			for _, host := range []string{"www.cloudflare.com", "api.ipify.org"} {
				r.Controls = append(r.Controls, residential.Observation{Scope: residential.Scope{Variant: v.Name, Domain: host}, At: now, Wave: wave, TLSVerified: true, OriginVerified: true, Outcome: "passed"})
			}
			for _, path := range v.Paths {
				for _, host := range j.Policy.Domains {
					for _, family := range j.Policy.Families {
						for _, transport := range j.Policy.Transports {
							r.Observations = append(r.Observations, residential.Observation{Scope: residential.Scope{Variant: v.Name, Domain: host, Family: family, Transport: transport, Path: path}, At: now, Wave: wave, ConnectionID: fmt.Sprintf("fixture-connection-%s-%d", v.Name, wave), Nonce: "fixture-unique-nonce-1234567890", IP: "8.8.8.8", Method: "server-flow-ed25519-v1", ConfigHash: v.SHA256, RuntimeHash: v.SHA256, PolicyHash: j.Policy.Digest(), Capability: "fixture-attestation", TLSVerified: true, OriginVerified: true, MeaningVerified: true, NonceBound: true, SameConnection: true, Connections: 1, ActualDomainRequest: true, ResponseSHA256: strings.Repeat("b", 64), Outcome: "passed", Code: "fixture_verified"})
						}
					}
				}
			}
		}
	}
	for _, source := range []string{"fixture-specialist", "fixture-independent"} {
		r.IPFacts = append(r.IPFacts, residential.IPFact{IP: "8.8.8.8", Source: source, At: now, Expires: now + 600000, Class: "Residential", EvidenceSHA256: strings.Repeat("c", 64)})
	}
	for i := range r.Observations {
		r.Observations[i].Attestation = base64.StdEncoding.EncodeToString(ed25519.Sign(ed25519.NewKeyFromSeed([]byte(strings.Repeat("s", 32))), residential.SigningPayload(r.Observations[i])))
	}
	for i := range r.IPFacts {
		r.IPFacts[i].MAC = residential.IPFactMAC(r.IPFacts[i], bytes.Repeat([]byte{7}, 32))
	}
	return residential.Completion{Hash: j.Hash, Token: j.Token, Revision: j.Revision, Epoch: j.Epoch, TrustHash: j.TrustHash, Candidate: plan.Variants[1].Config, Report: r}
}
func resSeed(t *testing.T, s *Store, n int) (string, *residential.Job, residential.Completion) {
	t.Helper()
	raw := strings.Replace(conversionSample, "192.0.2.10", fmt.Sprintf("192.0.2.%d", n), 1)
	source, j := addConversion(t, s, raw)
	if e := s.CompleteProbe(context.Background(), conversionResult(j, 0, true)); e != nil {
		t.Fatal(e)
	}
	job, e := s.ClaimResidential(context.Background())
	if e != nil || job == nil {
		t.Fatal("res claim", e)
	}
	result := resCompletion(*job)
	if e = s.CompleteResidential(context.Background(), result); e != nil {
		t.Fatal("res complete", e)
	}
	return source, job, result
}
func resRequest(s *Store, scope, format string) *httptest.ResponseRecorder {
	server, _ := NewServer(s, "http://127.0.0.1:18191")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, httptest.NewRequest("GET", server.residentialURL(scope, format), nil))
	return w
}
func TestResidentialRetentionTTLSourceDeletionAndGC(t *testing.T) {
	s := testStore(t)
	resPolicy(t, s)
	settings, _ := s.Settings(context.Background())
	settings.RetentionSeconds = 36000
	if e := s.SaveSettings(context.Background(), settings); e != nil {
		t.Fatal(e)
	}
	source, j, result := resSeed(t, s, 1)
	ctx := context.Background()
	if _, e := s.db.Exec("UPDATE artifacts SET created=created-10800000,expires=expires-10800000; UPDATE residential_records SET first_seen=first_seen-10800000,retain_until=retain_until-10800000"); e != nil {
		t.Fatal(e)
	}
	if e := s.Expire(ctx, time.Now()); e != nil {
		t.Fatal(e)
	}
	if e := s.Delete(ctx, source, 1); e != nil {
		t.Fatal(e)
	}
	if e := s.vault.Sweep(ctx, s.Referenced); e != nil {
		t.Fatal(e)
	}
	for _, hash := range []string{j.Hash, residential.Digest(result.Candidate)} {
		if used, e := s.Referenced(ctx, hash); e != nil || !used {
			t.Fatalf("retention %s %v %v", hash, used, e)
		}
	}
	for _, format := range []string{"configs.jsonl", "next.json", "raw.txt"} {
		w := resRequest(s, "all", format)
		if w.Code != 200 {
			t.Fatal(format, w.Code, w.Body.String())
		}
	}
	if _, e := s.db.Exec("UPDATE residential_records SET valid_until=0"); e != nil {
		t.Fatal(e)
	}
	for _, format := range []string{"configs.jsonl", "next.json", "raw.txt"} {
		w := resRequest(s, "all", format)
		if w.Code != 503 {
			t.Fatal("expired publication", format, w.Code)
		}
	}
	if used, _ := s.Referenced(ctx, j.Hash); !used {
		t.Fatal("quarantine lost original")
	}
}
func TestResidentialExportParityExactRawAndConcurrentRounds(t *testing.T) {
	s := testStore(t)
	resPolicy(t, s)
	expected := map[string][]byte{}
	for n := 1; n <= 4; n++ {
		_, job, _ := resSeed(t, s, n)
		expected[job.Hash] = job.Raw
	}
	a := resRequest(s, "all", "configs.jsonl")
	b := resRequest(s, "all", "raw.txt")
	if a.Code != 200 || b.Code != 200 || a.Header().Get("X-Residential-Generation") != b.Header().Get("X-Residential-Generation") {
		t.Fatal(a.Code, a.Body.String(), b.Code, b.Body.String())
	}
	lines := bytes.Split(bytes.TrimSuffix(a.Body.Bytes(), []byte{'\n'}), []byte{'\n'})
	if len(lines) != 4 {
		t.Fatal("jsonl member count", len(lines))
	}
	for _, line := range lines {
		if !json.Valid(line) || !bytes.Contains(line, []byte(`"remarks":"@shenvpn"`)) {
			t.Fatal("jsonl envelope")
		}
	}
	keys := make([]string, 0, len(expected))
	for hash := range expected {
		keys = append(keys, hash)
	}
	sort.Strings(keys)
	var rawExpected bytes.Buffer
	for i, hash := range keys {
		if i > 0 {
			rawExpected.WriteByte('\n')
		}
		rawExpected.Write(expected[hash])
	}
	if !bytes.Equal(b.Body.Bytes(), rawExpected.Bytes()) {
		t.Fatal("raw bytes or ordering changed")
	}
	if b.Header().Get("Content-Disposition") != "inline" || !strings.HasPrefix(b.Header().Get("Content-Type"), "text/plain") {
		t.Fatal("raw did not print inline")
	}
	other, e := Open(filepath.Join(filepath.Dir(s.vault.Directory()), "panel.db"), s.key)
	if e != nil {
		t.Fatal(e)
	}
	defer other.Close()
	responses := make(chan *httptest.ResponseRecorder, 16)
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		store := s
		if i%2 == 1 {
			store = other
		}
		go func() { defer wg.Done(); responses <- resRequest(store, "all", "next.json") }()
	}
	wg.Wait()
	close(responses)
	rounds := map[string]map[string]bool{}
	for w := range responses {
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		cycle := w.Header().Get("X-Export-Cycle")
		hash := w.Header().Get("X-Original-SHA256")
		if rounds[cycle] == nil {
			rounds[cycle] = map[string]bool{}
		}
		if rounds[cycle][hash] || expected[hash] == nil {
			t.Fatal("duplicate or foreign allocation", cycle, hash)
		}
		rounds[cycle][hash] = true
	}
	if len(rounds) != 4 {
		t.Fatal("round count", len(rounds))
	}
	for _, members := range rounds {
		if len(members) != 4 {
			t.Fatal("incomplete round")
		}
	}
}
func TestResidentialRevocationStaleJobsAndBackupReplay(t *testing.T) {
	s := testStore(t)
	resPolicy(t, s)
	_, job, _ := resSeed(t, s, 1)
	ctx := context.Background()
	if _, e := s.db.Exec("UPDATE residential_records SET next_run=0"); e != nil {
		t.Fatal(e)
	}
	stale, e := s.ClaimResidential(ctx)
	if e != nil || stale == nil {
		t.Fatal(e)
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.revokeResidential(ctx, tx, job.Hash, "verified_nonresidential_ads_sample", false, time.Now().UnixMilli()); e != nil {
		tx.Rollback()
		t.Fatal(e)
	}
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
	if e = s.CompleteResidential(ctx, resCompletion(*stale)); !errors.Is(e, ErrConflict) {
		t.Fatal("stale completion", e)
	}
	if rows, e := s.Conversions(ctx, 0, true); e != nil || len(rows) != 0 {
		t.Fatal("ordinary resurrected", e, len(rows))
	}
	if used, e := s.Referenced(ctx, job.Hash); e != nil || used {
		t.Fatal("deleted raw reference", e, used)
	}
	if e = s.residentialBarrier(ctx); e != nil {
		t.Fatal(e)
	}
	// Simulate restoring a pre-revocation DB sequence. Journal remains authoritative.
	if _, e = s.db.Exec("DELETE FROM residential_tombstones;DELETE FROM residential_identity_tombstones;UPDATE residential_settings SET journal_seq=0"); e != nil {
		t.Fatal(e)
	}
	if e = s.residentialBarrier(ctx); !errors.Is(e, errResidentialJournal) {
		t.Fatal("old backup not fenced", e)
	}
	path := filepath.Join(filepath.Dir(s.vault.Directory()), "panel.db")
	if e = s.initResidentialJournal(ctx, path); e != nil {
		t.Fatal("journal replay", e)
	}
	if e = s.residentialBarrier(ctx); e != nil {
		t.Fatal(e)
	}
	var state string
	if e = s.db.QueryRow("SELECT state FROM residential_records WHERE hash=?", job.Hash).Scan(&state); e != nil || state != "DELETED" {
		t.Fatal(state, e)
	}
	// A missing journal is an error, never a blank replacement.
	if e = os.Rename(s.residentialJournal, s.residentialJournal+".saved"); e != nil {
		t.Fatal(e)
	}
	if e = s.residentialBarrier(ctx); e == nil {
		t.Fatal("missing journal bypass")
	}
	if e = s.initResidentialJournal(ctx, path); e == nil {
		t.Fatal("missing journal silently initialized")
	}
}
func TestResidentialUnknownConflictOutageNeverDeletes(t *testing.T) {
	for _, mode := range []string{"expired-intel", "conflict", "outage", "missing-ip", "partial-ipv6"} {
		t.Run(mode, func(t *testing.T) {
			s := testStore(t)
			p := resPolicy(t, s)
			_, job, _ := resSeed(t, s, 1)
			ctx := context.Background()
			s.db.Exec("UPDATE residential_records SET next_run=0")
			next, e := s.ClaimResidential(ctx)
			if e != nil || next == nil {
				t.Fatal(e)
			}
			r := resCompletion(*next)
			switch mode {
			case "expired-intel":
				r.Report.IPFacts[0].Expires = 1
			case "conflict":
				r.Report.IPFacts[0].Class = "Datacenter"
			case "outage":
				r.Report.InfrastructureOK = false
			case "missing-ip":
				r.Report.Observations = nil
			case "partial-ipv6":
				p.Families = []int{4, 6}
				pb, _ := json.Marshal(p)
				s.db.Exec("UPDATE residential_settings SET policy=?,policy_hash=?", string(pb), p.Digest())
				r.Report.PolicyHash = p.Digest()
			}
			if e = s.CompleteResidential(ctx, r); e != nil {
				t.Fatal(e)
			}
			if w := resRequest(s, "all", "configs.jsonl"); w.Code != 503 {
				t.Fatal("uncertain publication")
			}
			var state string
			var n int
			s.db.QueryRow("SELECT state FROM residential_records WHERE hash=?", job.Hash).Scan(&state)
			s.db.QueryRow("SELECT count(*) FROM residential_tombstones").Scan(&n)
			if state != "QUARANTINED" || n != 0 {
				t.Fatal("uncertainty deletes", state, n)
			}
		})
	}
}
