package panel

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rezajafari0970/Config_manager/internal/repair"
	"github.com/rezajafari0970/Config_manager/internal/residential"
)

type ResidentialEntry struct {
	Hash          string `json:"hash"`
	CandidateHash string `json:"candidate_hash"`
	Kind          string `json:"kind"`
	Size          int64  `json:"size"`
	Revision      int64  `json:"revision"`
	Epoch         int64  `json:"epoch"`
	Tested        int64  `json:"tested"`
	ValidUntil    int64  `json:"valid_until"`
	RetainUntil   int64  `json:"retain_until"`
}
type ResidentialSnapshot struct {
	Generation      string             `json:"generation"`
	PolicyHash      string             `json:"policy_hash"`
	TrustHash       string             `json:"trust_hash"`
	JournalSequence int64              `json:"journal_sequence"`
	Entries         []ResidentialEntry `json:"entries"`
}
type residentialQuery interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func residentialSnapshot(ctx context.Context, q residentialQuery, scope string) (ResidentialSnapshot, error) {
	v := ResidentialSnapshot{Entries: []ResidentialEntry{}}
	if e := q.QueryRowContext(ctx, "SELECT policy_hash,trust_hash,journal_seq FROM residential_settings WHERE id=1").Scan(&v.PolicyHash, &v.TrustHash, &v.JournalSequence); e != nil {
		return v, e
	}
	view := "residential_eligible"
	if strings.HasPrefix(scope, "ads:") {
		view = "ads_different_eligible"
		scope = strings.TrimPrefix(scope, "ads:")
	}
	query := `SELECT hash,candidate_hash,kind,size,revision,epoch,tested,valid_until,retain_until FROM ` + view + ` e`
	args := []any{}
	if scope != "all" {
		query += ` WHERE EXISTS(SELECT 1 FROM residential_owners o WHERE o.hash=e.hash AND o.source=?)`
		args = append(args, scope)
	}
	rows, e := q.QueryContext(ctx, query+" ORDER BY hash", args...)
	if e != nil {
		return v, e
	}
	defer rows.Close()
	for rows.Next() {
		var r ResidentialEntry
		if e = rows.Scan(&r.Hash, &r.CandidateHash, &r.Kind, &r.Size, &r.Revision, &r.Epoch, &r.Tested, &r.ValidUntil, &r.RetainUntil); e != nil {
			return v, e
		}
		v.Entries = append(v.Entries, r)
	}
	if e = rows.Err(); e != nil {
		return v, e
	}
	b, _ := json.Marshal(v)
	v.Generation = residential.Digest(b)
	return v, nil
}
func (s *Store) ResidentialSnapshot(ctx context.Context, scope string) (ResidentialSnapshot, error) {
	tx, e := s.reader().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return ResidentialSnapshot{}, e
	}
	defer tx.Rollback()
	v, e := residentialSnapshot(ctx, tx, scope)
	if e != nil {
		return v, e
	}
	return v, tx.Commit()
}
func (s *Store) residentialBytes(hash string, compact bool) ([]byte, error) {
	body, e := s.vault.Open(hash)
	if e != nil {
		return nil, e
	}
	defer body.Close()
	b, e := io.ReadAll(io.LimitReader(body, repair.MaxBytes+1))
	if e != nil {
		return nil, e
	}
	if len(b) > repair.MaxBytes || repair.Digest(b) != hash {
		return nil, ErrInvalid
	}
	if compact {
		var out bytes.Buffer
		if json.Compact(&out, b) != nil || !bytes.Equal(out.Bytes(), b) || !bytes.HasPrefix(b, []byte("{")) {
			return nil, ErrInvalid
		}
	}
	return b, nil
}
func (s *Server) residentialURL(scope, format string) string {
	path, externalScope := "/residential-subscription/", scope
	if strings.HasPrefix(scope, "ads:") {
		path = "/ads-different-subscription/"
		externalScope = strings.TrimPrefix(scope, "ads:")
	}
	return s.origin + path + externalScope + "/" + s.store.digest("residential-export-v1", scope+":"+format) + "/" + format
}

type residentialModule struct{ s *Server }

func (m residentialModule) Register(mux *http.ServeMux) {
	s := m.s
	mux.HandleFunc("POST /api/residential/manage", s.residentialManage)
	mux.HandleFunc("GET /api/residential", s.residentialState)
	mux.HandleFunc("GET /api/residential/evidence/{hash}", s.residentialEvidence)
	mux.HandleFunc("GET /residential-subscription/{scope}/{token}/raw.zip", func(w http.ResponseWriter, r *http.Request) {
		if !equal(r.PathValue("token"), s.store.digest("residential-export-v1", r.PathValue("scope")+":raw.zip")) {
			http.NotFound(w, r)
			return
		}
		target := s.residentialURL(r.PathValue("scope"), "raw.txt")
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		w.Header().Set("Cache-Control", "no-store")
		http.Redirect(w, r, target, http.StatusPermanentRedirect)
	})
	for _, format := range []string{"configs.jsonl", "next.json", "raw.txt"} {
		mux.HandleFunc("GET /residential-subscription/{scope}/{token}/"+format, s.residentialExport)
		mux.HandleFunc("GET /ads-different-subscription/{scope}/{token}/"+format, s.residentialExport)
	}
}
func (s *Server) residentialExport(w http.ResponseWriter, r *http.Request) {
	scope := r.PathValue("scope")
	if strings.HasPrefix(r.URL.Path, "/ads-different-subscription/") {
		scope = "ads:" + scope
	}
	format := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	if !equal(r.PathValue("token"), s.store.digest("residential-export-v1", scope+":"+format)) {
		http.NotFound(w, r)
		return
	}
	if e := s.store.residentialBarrier(r.Context()); e != nil {
		http.Error(w, "revocation state unavailable", 503)
		return
	}
	snapshot, e := s.store.ResidentialSnapshot(r.Context(), scope)
	if e != nil {
		failure(w, e)
		return
	}
	w.Header().Set("X-Residential-Generation", snapshot.Generation)
	w.Header().Set("X-Evidence-Class", "verified-residential-within-policy")
	if strings.HasPrefix(scope, "ads:") {
		w.Header().Set("X-Evidence-Class", "different-ads-egress; not-residential-classification")
	}
	w.Header().Set("X-Config-Count", strconv.Itoa(len(snapshot.Entries)))
	w.Header().Set("X-Verification-Scope", "explicit-domains-and-protocols-only; observed-not-guaranteed")
	if requested := r.URL.Query().Get("generation"); requested != "" && requested != snapshot.Generation {
		http.Error(w, "generation changed; refresh all three links", 409)
		return
	}
	if len(snapshot.Entries) == 0 {
		w.Header().Set("Retry-After", "15")
		http.Error(w, "no currently qualified configuration in this evidence class", 503)
		return
	}
	if r.Method == http.MethodHead {
		w.WriteHeader(200)
		return
	}
	if strings.Contains(strings.ToLower(r.Header.Get("Sec-Purpose")+r.Header.Get("Purpose")), "prefetch") {
		w.WriteHeader(204)
		return
	}
	if format == "next.json" {
		s.residentialNext(w, r, scope, snapshot)
		return
	}
	spool, e := newExportSpool()
	if e != nil {
		failure(w, e)
		return
	}
	defer spool.Close()
	rawOutput := format == "raw.txt"
	for i, entry := range snapshot.Entries {
		hash := entry.CandidateHash
		if rawOutput {
			hash = entry.Hash
		}
		b, err := s.store.residentialBytes(hash, !rawOutput)
		if err != nil {
			s.store.quarantineResidentialBlob(r.Context(), entry)
			http.Error(w, "member requires revalidation", 503)
			return
		}
		if rawOutput {
			writer := io.MultiWriter(spool.file, spool.digest)
			if i > 0 {
				_, e = writer.Write([]byte{'\n'})
			}
			if e == nil {
				_, e = writer.Write(b)
			}
		} else {
			e = spool.Line(b)
		}
		if e != nil {
			failure(w, e)
			return
		}
	}
	// Admission is the linearization point. After admission the response is in
	// flight; it cannot be recalled even if a client has not yet received headers.
	tx, e := s.store.db.BeginTx(r.Context(), nil)
	if e != nil {
		failure(w, e)
		return
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(r.Context(), "UPDATE residential_settings SET generation=generation WHERE id=1"); e != nil {
		failure(w, e)
		return
	}
	fresh, e := residentialSnapshot(r.Context(), tx, scope)
	if e != nil || fresh.Generation != snapshot.Generation {
		http.Error(w, "membership or evidence changed", 503)
		return
	}
	if e = s.store.checkResidentialJournalTx(r.Context(), tx); e != nil {
		http.Error(w, "revocation pending", 503)
		return
	}
	if e = tx.Commit(); e != nil {
		failure(w, e)
		return
	}
	if e = spool.Rewind(); e != nil {
		failure(w, e)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("X-Content-SHA256", spool.Hash())
	if rawOutput {
		w.Header().Set("X-Raw-Framing", "original-bytes; one-LF-between-members")
	}
	_, _ = spool.WriteTo(w)
}

func (s *Server) residentialNext(w http.ResponseWriter, r *http.Request, scope string, snapshot ResidentialSnapshot) {
	// Blob I/O precedes the writer transaction. At most one candidate is loaded
	// per retry; cross-process turn competition does not use an arbitrary cap.
	for r.Context().Err() == nil {
		tx, e := s.store.db.BeginTx(r.Context(), nil)
		if e != nil {
			failure(w, e)
			return
		}
		_, e = tx.ExecContext(r.Context(), "INSERT INTO residential_rotation(scope) VALUES(?) ON CONFLICT DO NOTHING", scope)
		if e == nil {
			_, e = tx.ExecContext(r.Context(), "UPDATE residential_rotation SET cycle=cycle WHERE scope=?", scope)
		}
		fresh, err := residentialSnapshot(r.Context(), tx, scope)
		if e == nil {
			e = err
		}
		if e != nil {
			tx.Rollback()
			failure(w, e)
			return
		}
		if len(fresh.Entries) == 0 {
			tx.Rollback()
			http.Error(w, "no currently qualified configuration in this evidence class", 503)
			return
		}
		if requested := r.URL.Query().Get("generation"); requested != "" && requested != fresh.Generation {
			tx.Rollback()
			http.Error(w, "generation changed", 409)
			return
		}
		ids := []string{}
		byHash := map[string]ResidentialEntry{}
		for _, entry := range fresh.Entries {
			ids = append(ids, entry.Hash)
			byHash[entry.Hash] = entry
		}
		b, _ := json.Marshal(ids)
		if _, e = tx.ExecContext(r.Context(), "DELETE FROM residential_turns WHERE scope=? AND hash NOT IN (SELECT value FROM json_each(?))", scope, string(b)); e != nil {
			tx.Rollback()
			failure(w, e)
			return
		}
		var remaining int
		if e = tx.QueryRowContext(r.Context(), "SELECT count(*) FROM residential_turns WHERE scope=?", scope).Scan(&remaining); e != nil {
			tx.Rollback()
			failure(w, e)
			return
		}
		if remaining == 0 {
			if _, e = tx.ExecContext(r.Context(), "UPDATE residential_rotation SET cycle=cycle+1 WHERE scope=?", scope); e == nil {
				_, e = tx.ExecContext(r.Context(), "INSERT INTO residential_turns SELECT ?,CAST(key AS INTEGER),value FROM json_each(?)", scope, string(b))
			}
			if e != nil {
				tx.Rollback()
				failure(w, e)
				return
			}
		}
		var hash string
		var cycle int64
		if e = tx.QueryRowContext(r.Context(), "SELECT q.hash,s.cycle FROM residential_turns q JOIN residential_rotation s ON s.scope=q.scope WHERE q.scope=? ORDER BY q.position LIMIT 1", scope).Scan(&hash, &cycle); e != nil {
			tx.Rollback()
			failure(w, e)
			return
		}
		if e = tx.Commit(); e != nil {
			failure(w, e)
			return
		}
		entry := byHash[hash]
		data, e := s.store.residentialBytes(entry.CandidateHash, true)
		if e != nil {
			s.store.quarantineResidentialBlob(r.Context(), entry)
			http.Error(w, "member requires revalidation", 503)
			return
		}
		tx, e = s.store.db.BeginTx(r.Context(), nil)
		if e != nil {
			failure(w, e)
			return
		}
		if _, e = tx.ExecContext(r.Context(), "UPDATE residential_rotation SET cycle=cycle WHERE scope=?", scope); e != nil {
			tx.Rollback()
			failure(w, e)
			return
		}
		current, e := residentialSnapshot(r.Context(), tx, scope)
		var head string
		var currentCycle int64
		if e == nil {
			e = tx.QueryRowContext(r.Context(), "SELECT q.hash,s.cycle FROM residential_turns q JOIN residential_rotation s ON s.scope=q.scope WHERE q.scope=? ORDER BY q.position LIMIT 1", scope).Scan(&head, &currentCycle)
		}
		if errors.Is(e, sql.ErrNoRows) || e == nil && (current.Generation != fresh.Generation || head != hash || cycle != currentCycle) {
			tx.Rollback()
			continue
		}
		if e == nil {
			e = s.store.checkResidentialJournalTx(r.Context(), tx)
		}
		if e == nil {
			_, e = tx.ExecContext(r.Context(), "DELETE FROM residential_turns WHERE scope=? AND hash=?", scope, hash)
		}
		if e != nil {
			tx.Rollback()
			failure(w, e)
			return
		}
		if e = tx.Commit(); e != nil {
			failure(w, e)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Residential-Generation", current.Generation)
		w.Header().Set("X-Config-Count", "1")
		w.Header().Set("X-Eligible-Count", strconv.Itoa(len(current.Entries)))
		w.Header().Set("X-Export-Cycle", strconv.FormatInt(cycle, 10))
		w.Header().Set("X-Original-SHA256", hash)
		w.Header().Set("X-Content-SHA256", entry.CandidateHash)
		w.Write(data)
		return
	}
}
func (s *Store) quarantineResidentialBlob(ctx context.Context, entry ResidentialEntry) {
	_, _ = s.db.ExecContext(ctx, `UPDATE residential_records SET state='QUARANTINED',reason='blob_unavailable',valid_until=0,next_run=0,revision=revision+1,token='',lease_until=0 WHERE hash=? AND revision=? AND candidate_hash=?`, entry.Hash, entry.Revision, entry.CandidateHash)
}
func (s *Store) checkResidentialJournalTx(ctx context.Context, tx *sql.Tx) error {
	f, e := s.openRevocations(false)
	if e != nil {
		return e
	}
	defer f.Close()
	last, e := s.lastRevocation(f)
	if e != nil {
		return e
	}
	mark, e := s.readRevocationMark()
	if e != nil || mark.Sequence != last.Sequence || mark.MAC != last.MAC {
		return errResidentialJournal
	}
	var seq int64
	if e = tx.QueryRowContext(ctx, "SELECT journal_seq FROM residential_settings WHERE id=1").Scan(&seq); e != nil {
		return e
	}
	if seq != last.Sequence {
		return errResidentialJournal
	}
	return nil
}

func (s *Server) residentialEvidence(w http.ResponseWriter, r *http.Request) {
	if !validDigest(r.PathValue("hash")) {
		failure(w, ErrInvalid)
		return
	}
	rows, e := s.store.reader().QueryContext(r.Context(), "SELECT at,state,reason,report,decision FROM residential_audit WHERE hash=? ORDER BY id DESC LIMIT 20", r.PathValue("hash"))
	if e != nil {
		failure(w, e)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var at int64
		var state, reason, report, decision string
		if e = rows.Scan(&at, &state, &reason, &report, &decision); e != nil {
			failure(w, e)
			return
		}
		out = append(out, map[string]any{"at": at, "state": state, "reason": reason, "report": json.RawMessage(report), "decision": json.RawMessage(decision)})
	}
	if e = rows.Err(); e != nil {
		failure(w, e)
		return
	}
	respond(w, 200, out)
}
func (s *Server) residentialState(w http.ResponseWriter, r *http.Request) {
	scope := r.URL.Query().Get("source")
	if scope == "" {
		scope = "all"
	}
	if !safeCode(scope) {
		failure(w, ErrInvalid)
		return
	}
	snapshot, e := s.store.ResidentialSnapshot(r.Context(), scope)
	if e != nil {
		failure(w, e)
		return
	}
	p, tr, e := residentialSettings(r.Context(), s.store.reader())
	if e != nil {
		failure(w, e)
		return
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
	if offset > 1000000 {
		offset = 1000000
	}
	rows, e := s.store.reader().QueryContext(r.Context(), `SELECT hash,state,reason,ever_accepted,tested,valid_until,retain_until,next_run,decision FROM residential_records r WHERE (?='all' OR EXISTS(SELECT 1 FROM residential_owners o WHERE o.hash=r.hash AND o.source=?)) ORDER BY ever_accepted DESC,first_seen DESC,hash LIMIT 100 OFFSET ?`, scope, scope, offset)
	if e != nil {
		failure(w, e)
		return
	}
	entries := []map[string]any{}
	for rows.Next() {
		var hash, state, reason, decision string
		var retained bool
		var tested, until, retainedUntil, next int64
		if e = rows.Scan(&hash, &state, &reason, &retained, &tested, &until, &retainedUntil, &next, &decision); e != nil {
			rows.Close()
			failure(w, e)
			return
		}
		if !retained && tested > 0 && state != "DELETED" && retainedUntil <= time.Now().UnixMilli() {
			state = "DELETED"
			reason = "retention_expired"
		}
		if (state == "ACCEPTED" || state == "ADS_DIFFERENT") && until <= time.Now().UnixMilli() {
			state = "QUARANTINED"
			reason = "evidence_expired"
		}
		entries = append(entries, map[string]any{"hash": hash, "state": state, "reason": reason, "retained": retained, "tested": tested, "valid_until": until, "retain_until": retainedUntil, "retention_exempt": retained, "first_assessment_pending": tested == 0 && state != "DELETED", "next_run": next, "decision": json.RawMessage(decision)})
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		failure(w, e)
		return
	}
	counts := map[string]int{}
	rows, e = s.store.reader().QueryContext(r.Context(), "SELECT CASE WHEN ever_accepted=0 AND tested>0 AND state!='DELETED' AND retain_until<=CAST(unixepoch('subsec')*1000 AS INTEGER) THEN 'DELETED' WHEN state IN ('ACCEPTED','ADS_DIFFERENT') AND hash NOT IN (SELECT hash FROM residential_active) THEN 'QUARANTINED' ELSE state END AS visible,count(*) FROM residential_records r WHERE (?='all' OR EXISTS(SELECT 1 FROM residential_owners o WHERE o.hash=r.hash AND o.source=?)) GROUP BY visible", scope, scope)
	if e != nil {
		failure(w, e)
		return
	}
	for rows.Next() {
		var state string
		var n int
		if e = rows.Scan(&state, &n); e != nil {
			rows.Close()
			failure(w, e)
			return
		}
		counts[state] = n
	}
	rows.Close()
	var total, tested, healthy, pending int
	if e = s.store.reader().QueryRowContext(r.Context(), `SELECT count(*),coalesce(sum(tested>0),0),coalesce(sum(json_extract(decision,'$.health')='HEALTHY'),0),coalesce(sum(tested=0 AND state!='DELETED'),0) FROM residential_records r WHERE (?='all' OR EXISTS(SELECT 1 FROM residential_owners o WHERE o.hash=r.hash AND o.source=?))`, scope, scope).Scan(&total, &tested, &healthy, &pending); e != nil {
		failure(w, e)
		return
	}
	var worker probeStatus
	_ = s.store.reader().QueryRowContext(r.Context(), "SELECT status,code,heartbeat,active FROM residential_worker WHERE id=1").Scan(&worker.Status, &worker.Code, &worker.Heartbeat, &worker.Active)
	if time.Now().UnixMilli()-worker.Heartbeat > 30000 {
		worker.Status = "offline"
	}
	links := map[string]string{}
	for _, format := range []string{"configs.jsonl", "next.json", "raw.txt"} {
		links[format] = s.residentialURL(scope, format)
	}
	adsSnapshot, e := s.store.ResidentialSnapshot(r.Context(), "ads:"+scope)
	if e != nil {
		failure(w, e)
		return
	}
	adsLinks := map[string]string{}
	for _, format := range []string{"configs.jsonl", "next.json", "raw.txt"} {
		adsLinks[format] = s.residentialURL("ads:"+scope, format)
	}
	metrics, e := s.store.residentialMetrics(r.Context())
	if e != nil {
		failure(w, e)
		return
	}
	respond(w, 200, map[string]any{"ads_different": map[string]any{"eligible_count": len(adsSnapshot.Entries), "generation": adsSnapshot.Generation, "links": adsLinks}, "egress_trust_configured": len(tr.Capabilities) > 0, "offset": offset, "total": total, "progress": map[string]int{"tested": tested, "healthy": healthy, "pending": pending}, "metrics": metrics, "generation": snapshot.Generation, "eligible_count": len(snapshot.Entries), "policy": p, "trust_configured": len(tr.Capabilities) > 0 && len(tr.Providers) >= 2, "counts": counts, "entries": entries, "links": links, "worker": worker, "scope": scope, "now": time.Now().UnixMilli(), "note": "Evidence applies only to the explicit tested matrix, not all Google Ads or future sessions."})
}

type probeStatus struct {
	Status    string `json:"status"`
	Code      string `json:"code"`
	Heartbeat int64  `json:"heartbeat"`
	Active    int    `json:"active"`
}
