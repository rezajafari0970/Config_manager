package panel

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rezajafari0970/Config_manager/internal/collect"
)

func acquisitionResult(t *testing.T, s *Store, raws ...string) collect.Result {
	t.Helper()
	r := collect.Result{State: "updated", HTTPStatus: 200, ContentState: "config", Extracted: len(raws), ValidCount: len(raws)}
	ref, e := s.vault.Put(context.Background(), strings.NewReader(strings.Join(raws, "\n")))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.vault.Release(ref.Hash) })
	r.Raw = ref
	for _, raw := range raws {
		one := fixtureResult(t, s, raw)
		r.Items = append(r.Items, one.Items[0])
	}
	return r
}

func TestAcquisitionDeletedItemsAreNotSuccessOrDuplicates(t *testing.T) {
	for _, staged := range []bool{false, true} {
		t.Run(map[bool]string{false: "items", true: "staged"}[staged], func(t *testing.T) {
			s := testStore(t)
			ctx := context.Background()
			id, _ := s.Save(ctx, "", 0, testSource())
			r := acquisitionResult(t, s, "vless://one@vpn.example:443#same")
			if e := s.Complete(ctx, claimOne(t, s, id), r, time.Now()); e != nil {
				t.Fatal(e)
			}
			before, _ := s.Get(ctx, id)
			if _, e := s.ManageConfigs(ctx, ConfigBatch{Action: "delete", All: true}); e != nil {
				t.Fatal(e)
			}
			j := claimOne(t, s, id)
			if staged {
				if _, e := s.Stage(ctx, j, r.Items[0]); e != nil {
					t.Fatal(e)
				}
				r.Items = nil
				r.Staged = true
			}
			if e := s.Complete(ctx, j, r, time.Now()); e != nil {
				t.Fatal(e)
			}
			got, _ := s.Get(ctx, id)
			if got.State != "filtered" || got.ErrorCode != "configs_deleted" || got.ConfigCount != 0 || got.StoredCount != 0 || got.SuppressedCount != 1 || got.DuplicateCount != 0 || got.LastSuccess != before.LastSuccess {
				t.Fatalf("false acquisition success: %+v", got)
			}
			stats, _ := s.Stats(ctx)
			if stats.GivingConfigs != 0 {
				t.Fatal(stats)
			}
		})
	}
}

func TestAcquisitionSharedDuplicatesAndPartialSuppression(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	id, _ := s.Save(ctx, "", 0, testSource())
	r := acquisitionResult(t, s, "vless://one@vpn.example:443#one", "vless://two@vpn.example:443#two")
	if e := s.Complete(ctx, claimOne(t, s, id), r, time.Now()); e != nil {
		t.Fatal(e)
	}
	if _, e := s.ManageConfigs(ctx, ConfigBatch{Action: "delete", IDs: []string{r.Items[0].Identity}}); e != nil {
		t.Fatal(e)
	}
	other := testSource()
	other.URL = "https://example.net/other"
	b, _ := s.Save(ctx, "", 0, other)
	j := claimOne(t, s, b)
	for _, item := range r.Items {
		if _, e := s.Stage(ctx, j, item); e != nil {
			t.Fatal(e)
		}
	}
	r.Staged = true
	r.Items = nil
	r.Extracted = 3 // one repeated item, one existing, one deleted
	if e := s.Complete(ctx, j, r, time.Now()); e != nil {
		t.Fatal(e)
	}
	got, _ := s.Get(ctx, b)
	if got.State != "updated" || got.ConfigCount != 1 || got.StoredCount != 1 || got.SuppressedCount != 1 || got.DuplicateCount != 2 {
		t.Fatal(got)
	}
	stats, _ := s.Stats(ctx)
	if stats.Configs != 1 || stats.GivingConfigs != 2 {
		t.Fatal(stats)
	}
}

func TestRestoreSourceScopedFreshReacquisitionAndStaleWorkerFence(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	id, _ := s.Save(ctx, "", 0, testSource())
	r := acquisitionResult(t, s, "vless://one@vpn.example:443#one")
	if e := s.Complete(ctx, claimOne(t, s, id), r, time.Now()); e != nil {
		t.Fatal(e)
	}
	other := testSource()
	other.URL = "https://example.net/other"
	b, _ := s.Save(ctx, "", 0, other)
	rb := acquisitionResult(t, s, "vless://other@vpn.example:443#other")
	if e := s.Complete(ctx, claimOne(t, s, b), rb, time.Now()); e != nil {
		t.Fatal(e)
	}
	if _, e := s.ManageConfigs(ctx, ConfigBatch{Action: "delete", All: true}); e != nil {
		t.Fatal(e)
	}
	// A new ordinary receipt records suppression and supplies immutable recovery IDs.
	if e := s.Complete(ctx, claimOne(t, s, id), r, time.Now()); e != nil {
		t.Fatal(e)
	}
	old := claimOne(t, s, id)
	before, _ := s.Get(ctx, id)
	n, e := s.RestoreSource(ctx, id, before.Revision)
	if e != nil || n != 1 {
		t.Fatal(n, e)
	}
	if e = s.Complete(ctx, old, r, time.Now()); !errors.Is(e, ErrConflict) {
		t.Fatal("old job revived", e)
	}
	var controls int
	s.db.QueryRow("SELECT count(*) FROM config_controls WHERE id=?", rb.Items[0].Identity).Scan(&controls)
	if controls != 1 {
		t.Fatal("unrelated deletion cleared")
	}
	got, _ := s.Get(ctx, id)
	if got.ConfigCount != 0 || got.Revision != before.Revision+1 {
		t.Fatal("cached output restored without collection", got)
	}
	fresh := claimOne(t, s, id)
	if fresh.ETag != "" || fresh.Modified != "" {
		t.Fatal("recovery reused validators")
	}
	if e = s.Complete(ctx, fresh, r, time.Now()); e != nil {
		t.Fatal(e)
	}
	got, _ = s.Get(ctx, id)
	if got.ConfigCount != 1 || got.SuppressedCount != 0 || got.StoredCount != 1 {
		t.Fatal(got)
	}
	art, e := s.Output(ctx, r.Items[0].Identity)
	if e != nil || art.Hash != r.Items[0].Hash {
		t.Fatal("raw bytes changed", e)
	}
	// An explicit deletion after restoration still wins over a fetch in flight.
	inflight := claimOne(t, s, id)
	s.ManageConfigs(ctx, ConfigBatch{Action: "delete", IDs: []string{art.ID}})
	if e = s.Complete(ctx, inflight, r, time.Now()); e != nil {
		t.Fatal(e)
	}
	got, _ = s.Get(ctx, id)
	if got.ConfigCount != 0 || got.State != "filtered" {
		t.Fatal("later deletion ignored", got)
	}
}

func TestRestoreRejectsStaleDisabledExpiredAndCancelledRequests(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	id, _ := s.Save(ctx, "", 0, testSource())
	r := acquisitionResult(t, s, "vless://one@vpn.example:443#one")
	s.Complete(ctx, claimOne(t, s, id), r, time.Now())
	s.ManageConfigs(ctx, ConfigBatch{Action: "delete", IDs: []string{r.Items[0].Identity}})
	got, _ := s.Get(ctx, id)
	if _, e := s.RestoreSource(ctx, id, got.Revision+1); !errors.Is(e, ErrConflict) {
		t.Fatal(e)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, e := s.RestoreSource(cancelled, id, got.Revision); e == nil {
		t.Fatal("cancelled request restored")
	}
	in := got.SourceInput
	in.Enabled = false
	s.Save(ctx, id, got.Revision, in)
	if _, e := s.RestoreSource(ctx, id, got.Revision+1); !errors.Is(e, ErrConflict) {
		t.Fatal(e)
	}
	in.Enabled = true
	s.Save(ctx, id, got.Revision+1, in)
	s.db.Exec("UPDATE responses SET expires=0 WHERE source_id=?", id)
	if _, e := s.RestoreSource(ctx, id, got.Revision+2); !errors.Is(e, ErrNotFound) {
		t.Fatal(e)
	}
	var n int
	s.db.QueryRow("SELECT count(*) FROM config_controls").Scan(&n)
	if n != 1 {
		t.Fatal("failed restore changed controls")
	}
}

func TestUnchangedResponseWithoutLiveMembersIsNotSuccess(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	id, _ := s.Save(ctx, "", 0, testSource())
	r := acquisitionResult(t, s, "vless://one@vpn.example:443")
	s.Complete(ctx, claimOne(t, s, id), r, time.Now())
	before, _ := s.Get(ctx, id)
	j := claimOne(t, s, id)
	s.db.Exec("DELETE FROM artifacts")
	if e := s.Complete(ctx, j, collect.Result{State: "unchanged", HTTPStatus: 304}, time.Now()); e != nil {
		t.Fatal(e)
	}
	got, _ := s.Get(ctx, id)
	if got.State != "error" || got.ErrorCode != "cache_missing" || got.LastSuccess != before.LastSuccess {
		t.Fatal(got)
	}
	j = claimOne(t, s, id)
	if j.ETag != "" || j.Modified != "" {
		t.Fatal(j)
	}
}

func TestRestoreSourceHTTPRequiresSameOriginAndRevision(t *testing.T) {
	s := testStore(t)
	app, _ := NewServer(s, "https://panel.example")
	w := request(app.Handler(), "POST", "/api/sources/synthetic/restore", `{"revision":1}`, "", nil, "https://attacker.example")
	if w.Code != 403 {
		t.Fatal("restore escaped CSRF", w.Code)
	}
}
