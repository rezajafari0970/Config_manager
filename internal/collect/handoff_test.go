package collect

import (
	"bytes"
	"context"
	"github.com/rezajafari0970/Config_manager/internal/blob"
	"testing"
	"time"
)

type abandonedHandoffRepo struct {
	Repository
	job       Job
	abandoned []Job
}

func (r *abandonedHandoffRepo) Settings(context.Context) (Settings, error) {
	return Settings{Concurrency: 2, TimeoutSeconds: 10}, nil
}
func (r *abandonedHandoffRepo) Claim(context.Context, time.Time, int, time.Duration) ([]Job, error) {
	return []Job{r.job}, nil
}
func (r *abandonedHandoffRepo) Abandon(_ context.Context, j Job) error {
	r.abandoned = append(r.abandoned, j)
	return nil
}

func TestEditedSourceStillDrainingDoesNotStrandNewClaim(t *testing.T) {
	vault, e := blob.New(t.TempDir(), bytes.Repeat([]byte{3}, 32))
	if e != nil {
		t.Fatal(e)
	}
	repo := &abandonedHandoffRepo{job: Job{ID: "same-source", Token: "new-token", Revision: 2}}
	engine := NewEngine(repo, vault, nil)
	engine.active[repo.job.ID] = func() {} // Old canceled fetch has not finished its cleanup.
	engine.dispatch(context.Background())
	if len(repo.abandoned) != 1 || repo.abandoned[0].Token != "new-token" || repo.abandoned[0].Revision != 2 {
		t.Fatal("known discarded claim was left leased", repo.abandoned)
	}
	if len(engine.active) != 1 {
		t.Fatal("old active worker bookkeeping changed")
	}
}
