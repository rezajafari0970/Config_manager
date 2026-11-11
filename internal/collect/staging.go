package collect

import (
	"context"
	"github.com/rezajafari0970/Config_manager/internal/blob"
)

// Metadata batches amortize durable commits without retaining plaintext bodies.
// Every raw/candidate pin lasts through its batch commit or final error cleanup.
type stageBuffer struct {
	repo   Repository
	vault  *blob.Vault
	job    Job
	items  []Item
	pins   []string
	unique int
}

func newStageBuffer(repo Repository, vault *blob.Vault, job Job) *stageBuffer {
	return &stageBuffer{repo: repo, vault: vault, job: job}
}
func (b *stageBuffer) pin(hash string) { b.pins = append(b.pins, hash) }
func (b *stageBuffer) release() {
	for _, hash := range b.pins {
		b.vault.Release(hash)
	}
	b.pins = nil
}
func (b *stageBuffer) add(ctx context.Context, item Item) error {
	b.items = append(b.items, item)
	if len(b.items) >= 32 {
		return b.flush(ctx)
	}
	return nil
}
func (b *stageBuffer) flush(ctx context.Context) error {
	if len(b.items) == 0 {
		return nil
	}
	defer b.release()
	if repo, ok := b.repo.(BatchRepository); ok {
		n, err := repo.StageBatch(ctx, b.job, b.items)
		if err != nil {
			return err
		}
		b.unique += n
	} else {
		for _, item := range b.items {
			added, err := b.repo.Stage(ctx, b.job, item)
			if err != nil {
				return err
			}
			if added {
				b.unique++
			}
		}
	}
	clear(b.items)
	b.items = b.items[:0]
	return nil
}
