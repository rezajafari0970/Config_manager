package panel

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/rezajafari0970/Config_manager/internal/repair"
	"github.com/rezajafari0970/Config_manager/internal/smart"
)

type smartClaimInventory struct {
	pool   []smart.Member
	ids    []byte
	digest string
}

func smartClaimSnapshot(pool []smart.Member, ads bool) smartClaimInventory {
	ids := map[string]bool{}
	hashes := map[string]string{}
	for _, m := range pool {
		if old := hashes[m.Hash]; old != "" && old != m.ID {
			continue
		}
		hashes[m.Hash] = m.ID
		if !ads || m.Ads {
			ids[m.ID] = true
		}
	}
	// The same candidate has one canonical head, including follower offers.
	unique := make([]smart.Member, 0, len(pool))
	identity := make([][4]string, 0, len(pool))
	for _, m := range pool {
		if hashes[m.Hash] == m.ID {
			unique = append(unique, m)
			identity = append(identity, [4]string{m.ID, m.Source, m.RawHash, m.Hash})
		}
	}
	currentIDs := make([]string, 0, len(ids))
	for id := range ids {
		currentIDs = append(currentIDs, id)
	}
	encodedIDs, _ := json.Marshal(currentIDs)
	encodedInventory, _ := json.Marshal(identity)
	return smartClaimInventory{pool: unique, ids: encodedIDs, digest: repair.Digest(encodedInventory)}
}

// Idle Smart workers must not queue behind or occupy the sole writer. Inspect
// one WAL snapshot, including lease cleanup and inventory/expiry changes. Every
// positive path still repeats the authoritative checks inside ClaimSmart's
// writer transaction; a change after a negative read is seen by the next poll.
func (s *SmartPipeline) smartClaimNeeded(ctx context.Context) (bool, error) {
	readers := smartDB{DB: s.Store.workerReader(), splash: s.splash, ads: s.ads}
	tx, err := readers.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	var enabled bool
	var revision int64
	if err = tx.QueryRowContext(ctx, "SELECT enabled,revision FROM smart_settings WHERE id=1").Scan(&enabled, &revision); err != nil {
		return false, err
	}
	now := time.Now().UnixMilli()
	var invalid, due bool
	var active int
	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM smart_heads
 WHERE token!='' AND (version!=? OR revision!=? OR expires<=? OR
 (SELECT count(*) FROM smart_owned_members m WHERE m.head=smart_heads.id)!=member_count)),
 (SELECT count(*) FROM smart_heads WHERE token!='' AND lease_until>?),
 EXISTS(SELECT 1 FROM smart_heads WHERE (token='' OR lease_until<=?)
 AND (state!='ready' OR revision!=? OR version!=?) AND next_run<=?)`,
		s.version(), revision, now, now, now, revision, s.version(), now).Scan(&invalid, &active, &due)
	if err != nil || invalid {
		return invalid, err
	}
	if !enabled || active >= 2 {
		return false, nil
	}
	// Known due work already needs the authoritative writer path. Avoid building
	// its whole inventory twice; detailed read-only checks are for idle polls.
	if due {
		return true, nil
	}
	pool, err := smartInventory(ctx, tx)
	if err != nil {
		return false, err
	}
	snapshot := smartClaimSnapshot(pool, s.ads)
	var needed bool
	err = tx.QueryRowContext(ctx, `SELECT
 EXISTS(SELECT value FROM json_each(?) EXCEPT SELECT id FROM smart_heads)
 OR EXISTS(SELECT id FROM smart_heads EXCEPT SELECT value FROM json_each(?))
 OR EXISTS(SELECT 1 FROM smart_heads WHERE state='ready' AND id IN(SELECT id FROM smart_refresh_needed))
 OR EXISTS(SELECT 1 FROM smart_heads WHERE (token='' OR lease_until<=?)
 AND (state!='ready' OR revision!=? OR version!=?) AND (next_run<=? OR inventory!=?))`,
		string(snapshot.ids), string(snapshot.ids), now, revision, s.version(), now, snapshot.digest).Scan(&needed)
	return needed, err
}
