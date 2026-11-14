package panel

import (
	"bytes"
	"context"
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// A retained archive must neither become work again nor monopolize the panel
// when idle investigation workers poll while clients request individual JSON.
func TestSingleExportsDuringArchivedEgressPolling(t *testing.T) {
	s := testStore(t)
	source, expected := newExportSource(t, s, "single-contention", sourceURI(273), sourceURI(274))
	if _, err := s.db.Exec(`UPDATE egress_jobs SET state='NO_SPLIT',reason='non_ads_uses_ads_egress',tested=1,first_completed=1;
 WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i+1 FROM n WHERE i<30000)
 INSERT INTO egress_jobs(hash,kind,size,first_seen,last_seen) SELECT printf('%064x',100000+i),'fixture',1,1,1 FROM n;
 INSERT INTO egress_unique(identity,hash,created) SELECT hash,hash,1 FROM egress_jobs WHERE kind='fixture';`); err != nil {
		t.Fatal(err)
	}
	app, err := NewServer(s, "http://127.0.0.1:18191")
	if err != nil {
		t.Fatal(err)
	}
	h := app.Handler()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	start := make(chan struct{})
	errors := make(chan error, 14)
	var wg sync.WaitGroup
	for worker := 0; worker < 12; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			<-start
			for i := 0; i < 4; i++ {
				lane := []string{"", "initial", "followup"}[worker%3]
				job, err := s.ClaimEgressLane(ctx, lane)
				if err != nil || job != nil {
					errors <- fmt.Errorf("idle poll: job=%v error=%v", job != nil, err)
					return
				}
			}
		}(worker)
	}
	for _, scope := range []string{"", source} {
		wg.Add(1)
		go func(scope string) {
			defer wg.Done()
			<-start
			seen := map[string]bool{}
			for i := 0; i < 4; i++ {
				rr := httptest.NewRecorder()
				r := httptest.NewRequest("GET", app.exportURL(scope, true), nil).WithContext(ctx)
				h.ServeHTTP(rr, r)
				hash := rr.Header().Get("X-Content-SHA256")
				turn := rr.Header().Get("X-Export-Cycle") + ":" + hash
				if rr.Code != 200 || rr.Header().Get("X-Config-Count") != "1" || hash == "" || seen[turn] || !bytes.Equal(bytes.TrimSpace(rr.Body.Bytes()), expected[hash]) {
					errors <- fmt.Errorf("single export: status=%d count=%s repeated=%v", rr.Code, rr.Header().Get("X-Config-Count"), seen[turn])
					return
				}
				seen[turn] = true
			}
		}(scope)
	}
	close(start)
	wg.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
	var attempts int
	if err := s.db.QueryRow("SELECT sum(attempts) FROM egress_jobs").Scan(&attempts); err != nil || attempts != 0 {
		t.Fatal("archive or terminal negative was retested", attempts, err)
	}
}
