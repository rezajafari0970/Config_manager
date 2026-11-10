package panel

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

var errResidentialJournal = errors.New("residential revocation journal unavailable")

type revocation struct {
	Sequence   int64    `json:"sequence"`
	Hash       string   `json:"hash"`
	Identities []string `json:"identities"`
	At         int64    `json:"at"`
	Epoch      int64    `json:"epoch"`
	Reason     string   `json:"reason"`
	Manual     bool     `json:"manual"`
	Previous   string   `json:"previous"`
	MAC        string   `json:"mac"`
}

func (s *Store) revocationMAC(r revocation) string {
	r.MAC = ""
	b, _ := json.Marshal(r)
	h := hmac.New(sha256.New, s.key)
	h.Write([]byte("shen-revocation-v1\x00"))
	h.Write(b)
	return hex.EncodeToString(h.Sum(nil))
}
func (s *Store) openRevocations(create bool) (*os.File, error) {
	flags := os.O_RDWR | syscall.O_NOFOLLOW
	if create {
		flags |= os.O_CREATE
	}
	f, e := os.OpenFile(s.residentialJournal, flags, 0600)
	if e != nil {
		return nil, errResidentialJournal
	}
	info, e := f.Stat()
	if e != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		f.Close()
		return nil, errResidentialJournal
	}
	return f, nil
}
func (s *Store) lastRevocation(f *os.File) (revocation, error) {
	var r revocation
	info, e := f.Stat()
	if e != nil {
		return r, e
	}
	if info.Size() == 0 {
		return r, nil
	}
	n := min(info.Size(), 65536)
	b := make([]byte, n)
	if _, e = f.ReadAt(b, info.Size()-n); e != nil {
		return r, e
	}
	if b[len(b)-1] != '\n' {
		return r, errResidentialJournal
	}
	start := len(b) - 2
	for start >= 0 && b[start] != '\n' {
		start--
	}
	if start < 0 && n < info.Size() {
		return r, errResidentialJournal
	}
	if json.Unmarshal(b[start+1:len(b)-1], &r) != nil || !equal(r.MAC, s.revocationMAC(r)) {
		return r, errResidentialJournal
	}
	return r, nil
}
func (s *Store) initResidentialJournal(ctx context.Context, path string) error {
	s.residentialJournal = path + ".residential-revocations.jsonl"
	if p := os.Getenv("SHEN_RESIDENTIAL_JOURNAL"); p != "" {
		if !filepath.IsAbs(p) {
			return errResidentialJournal
		}
		s.residentialJournal = p
	}
	if e := os.MkdirAll(filepath.Dir(s.residentialJournal), 0700); e != nil {
		return e
	}
	// Every mutation and recovery takes the DB writer before the journal lock.
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, "UPDATE residential_settings SET generation=generation WHERE id=1"); e != nil {
		return e
	}
	var applied int64
	if e = tx.QueryRowContext(ctx, "SELECT journal_seq FROM residential_settings WHERE id=1").Scan(&applied); e != nil {
		return e
	}
	f, e := s.openRevocations(applied == 0)
	if e != nil {
		return e
	}
	defer f.Close()
	if e = syscall.Flock(int(f.Fd()), syscall.LOCK_EX); e != nil {
		return e
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	reader := bufio.NewReaderSize(f, 65536)
	var seq, validOffset int64
	previous := ""
	pending := []revocation{}
	chain := map[int64]string{0: ""}
	for {
		line, readErr := reader.ReadSlice('\n')
		if readErr == io.EOF && len(line) == 0 {
			break
		}
		if len(line) > 65536 {
			return errResidentialJournal
		}
		if readErr == io.EOF {
			// Only an uncommitted partial tail beyond the exact authenticated watermark
			// may be discarded. Missing committed history is never repaired as empty.
			mark, markErr := s.readRevocationMark()
			if markErr != nil || mark.Sequence != seq || mark.MAC != previous || applied > seq {
				return errResidentialJournal
			}
			if e = f.Truncate(validOffset); e != nil {
				return e
			}
			if e = f.Sync(); e != nil {
				return e
			}
			break
		}
		if readErr != nil {
			return errResidentialJournal
		}
		var r revocation
		if json.Unmarshal(line, &r) != nil || r.Sequence != seq+1 || r.Previous != previous || !equal(r.MAC, s.revocationMAC(r)) {
			return errResidentialJournal
		}
		validOffset += int64(len(line))
		seq = r.Sequence
		previous = r.MAC
		chain[seq] = r.MAC
		if seq > applied {
			pending = append(pending, r)
		}
	}
	if applied > seq {
		return errResidentialJournal
	}
	mark, e := s.readRevocationMark()
	if errors.Is(e, os.ErrNotExist) && seq == 0 && applied == 0 {
		e = s.writeRevocationMark(revocation{})
		mark = revocation{}
	}
	if e != nil {
		return errResidentialJournal
	}
	if mark.Sequence > seq || chain[mark.Sequence] != mark.MAC {
		return errResidentialJournal
	}
	// A crash after journal fsync but before the watermark is recoverable only
	// when the complete authenticated chain still contains the old watermark.
	if mark.Sequence < seq {
		if e = s.writeRevocationMark(revocation{Sequence: seq, MAC: previous}); e != nil {
			return e
		}
	}
	for _, r := range pending {
		if e = applyRevocation(ctx, tx, r); e != nil {
			return e
		}
	}
	if e = f.Sync(); e != nil {
		return e
	}
	dir, e := os.Open(filepath.Dir(s.residentialJournal))
	if e != nil {
		return e
	}
	defer dir.Close()
	if e = dir.Sync(); e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) readRevocationMark() (revocation, error) {
	var r revocation
	b, e := os.ReadFile(s.residentialJournal + ".highwater")
	if e != nil {
		return r, e
	}
	if len(b) > 1024 || json.Unmarshal(b, &r) != nil {
		return r, errResidentialJournal
	}
	return r, nil
}
func (s *Store) writeRevocationMark(r revocation) error {
	path := s.residentialJournal + ".highwater"
	f, e := os.CreateTemp(filepath.Dir(path), ".revocation-mark-")
	if e != nil {
		return e
	}
	name := f.Name()
	defer os.Remove(name)
	b, _ := json.Marshal(revocation{Sequence: r.Sequence, MAC: r.MAC})
	if _, e = f.Write(b); e == nil {
		e = f.Sync()
	}
	e = errors.Join(e, f.Close())
	if e != nil {
		return e
	}
	if e = os.Rename(name, path); e != nil {
		return e
	}
	dir, e := os.Open(filepath.Dir(path))
	if e != nil {
		return e
	}
	defer dir.Close()
	return dir.Sync()
}

// Publication stops during an uncommitted/replayed revocation. This also
// prevents a failed transaction or a restored older DB serving revoked bytes.
func (s *Store) residentialBarrier(ctx context.Context) error {
	f, e := s.openRevocations(false)
	if e != nil {
		return e
	}
	defer f.Close()
	if e = syscall.Flock(int(f.Fd()), syscall.LOCK_SH); e != nil {
		return e
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	r, e := s.lastRevocation(f)
	if e != nil {
		return e
	}
	mark, e := s.readRevocationMark()
	if e != nil || mark.Sequence != r.Sequence || mark.MAC != r.MAC {
		return errResidentialJournal
	}
	var applied int64
	if e = s.reader().QueryRowContext(ctx, "SELECT journal_seq FROM residential_settings WHERE id=1").Scan(&applied); e != nil {
		return e
	}
	if applied != r.Sequence {
		return errResidentialJournal
	}
	return nil
}
func (s *Store) revokeResidential(ctx context.Context, tx *sql.Tx, hash, reason string, manual bool, now int64) error {
	r := revocation{Hash: hash, At: now, Reason: reason, Manual: manual, Identities: []string{}}
	if e := tx.QueryRowContext(ctx, "SELECT epoch+1 FROM residential_records WHERE hash=?", hash).Scan(&r.Epoch); e != nil {
		return e
	}
	rows, e := tx.QueryContext(ctx, "SELECT DISTINCT identity FROM residential_owners WHERE hash=? ORDER BY identity", hash)
	if e != nil {
		return e
	}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return e
		}
		r.Identities = append(r.Identities, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	f, e := s.openRevocations(false)
	if e != nil {
		return e
	}
	defer f.Close()
	if e = syscall.Flock(int(f.Fd()), syscall.LOCK_EX); e != nil {
		return e
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	previous, e := s.lastRevocation(f)
	if e != nil {
		return e
	}
	var seq int64
	if e = tx.QueryRowContext(ctx, "SELECT journal_seq FROM residential_settings WHERE id=1").Scan(&seq); e != nil {
		return e
	}
	if seq != previous.Sequence {
		return errResidentialJournal
	}
	r.Sequence = seq + 1
	r.Previous = previous.MAC
	r.MAC = s.revocationMAC(r)
	b, _ := json.Marshal(r)
	if len(b) > 60000 {
		return ErrInvalid
	}
	if _, e = f.Seek(0, io.SeekEnd); e != nil {
		return e
	}
	if _, e = f.Write(append(b, '\n')); e != nil {
		return e
	}
	if e = f.Sync(); e != nil {
		return e
	}
	if e = s.writeRevocationMark(r); e != nil {
		return e
	}
	return applyRevocation(ctx, tx, r)
}
func applyRevocation(ctx context.Context, tx *sql.Tx, r revocation) error {
	identities, _ := json.Marshal(r.Identities)
	// Include every retained representation of an identity; old conversions,
	// source responses and staged work must not republish a removed variant.
	hashes := `SELECT hash FROM residential_owners WHERE identity IN (SELECT value FROM json_each(?)) UNION SELECT ?`
	if _, e := tx.ExecContext(ctx, `INSERT INTO residential_tombstones VALUES(?,?,?,?,?,?) ON CONFLICT(hash) DO UPDATE SET at=excluded.at,epoch=max(epoch,excluded.epoch),reason=excluded.reason,manual=max(manual,excluded.manual),seq=excluded.seq`, r.Hash, r.At, r.Epoch, r.Reason, r.Manual, r.Sequence); e != nil {
		// The table has six columns; keep the insert explicit to prevent migration drift.
		return e
	}
	for _, id := range r.Identities {
		if _, e := tx.ExecContext(ctx, `INSERT INTO residential_identity_tombstones VALUES(?,?,?,?) ON CONFLICT(identity) DO UPDATE SET at=excluded.at,hash=excluded.hash,manual=max(manual,excluded.manual)`, id, r.At, r.Hash, r.Manual); e != nil {
			return e
		}
	}
	for _, scope := range []string{"smart", "splash_smart", "ads_smart"} {
		for _, suffix := range []string{"cache", "heads"} {
			members := scope + "_cache_members"
			key := "id"
			if suffix == "heads" {
				members = scope + "_members"
			}
			q := "DELETE FROM " + scope + "_" + suffix + " WHERE " + key + " IN (SELECT head FROM " + members + " WHERE raw_hash IN (" + hashes + "))"
			if _, e := tx.ExecContext(ctx, q, string(identities), r.Hash); e != nil {
				return e
			}
		}
	}
	if _, e := tx.ExecContext(ctx, `DELETE FROM export_rotation_queue WHERE candidate_hash IN (SELECT candidate_hash FROM conversion_results WHERE hash IN (`+hashes+`))`, string(identities), r.Hash); e != nil {
		return e
	}
	for _, q := range []string{
		`DELETE FROM responses WHERE source_id IN (SELECT source_id FROM memberships WHERE artifact_id IN (SELECT value FROM json_each(?)))`,
		`UPDATE sources SET revision=revision+1,lease_token='',lease_until=0,etag='',modified='',refresh_reason='residential_revocation' WHERE id IN (SELECT source_id FROM memberships WHERE artifact_id IN (SELECT value FROM json_each(?)))`,
		`DELETE FROM staged WHERE identity IN (SELECT value FROM json_each(?))`,
		`DELETE FROM artifacts WHERE id IN (SELECT value FROM json_each(?))`,
	} {
		if _, e := tx.ExecContext(ctx, q, string(identities)); e != nil {
			return e
		}
	}
	for _, q := range []string{`DELETE FROM repairs WHERE raw_hash IN (` + hashes + `)`, `DELETE FROM conversion_jobs WHERE hash IN (` + hashes + `)`, `DELETE FROM residential_turns WHERE hash IN (` + hashes + `)`, `UPDATE residential_records SET state='DELETED',reason='confirmed_revocation',candidate_hash='',candidate_size=0,report='{}',decision='{}',valid_until=0,revision=revision+1,epoch=epoch+1,token='',lease_until=0 WHERE hash IN (` + hashes + `)`} {
		if _, e := tx.ExecContext(ctx, q, string(identities), r.Hash); e != nil {
			return e
		}
	}
	_, e := tx.ExecContext(ctx, "UPDATE residential_settings SET journal_seq=?,generation=generation+1,updated=? WHERE id=1", r.Sequence, r.At)
	return e
}
func (s *Store) recoverResidential(ctx context.Context) error {
	// A startup/restore requires fresh evidence even if an old backup's clock
	// deadlines still look current. Durable turns and retained originals survive.
	_, e := s.db.ExecContext(ctx, `UPDATE residential_records SET state=CASE WHEN ever_accepted=1 THEN 'QUARANTINED' ELSE state END,reason=CASE WHEN ever_accepted=1 THEN 'restart_requires_revalidation' ELSE reason END,valid_until=0,token='',lease_until=0,revision=revision+1,next_run=min(next_run,?) WHERE state!='DELETED'; UPDATE residential_settings SET generation=generation+1 WHERE id=1`, time.Now().UnixMilli())
	return e
}

// PrepareResidentialServing must run before either public or private listeners.
func (s *Store) PrepareResidentialServing(ctx context.Context) error {
	return s.recoverResidential(ctx)
}
