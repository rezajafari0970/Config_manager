package panel

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/convert"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"github.com/rezajafari0970/Config_manager/internal/repair"
	"io"
	"net/http"
	"strconv"
	"time"
)

type Conversion struct {
	ID            string `json:"id"`
	Hash          string `json:"hash"`
	Kind          string `json:"kind"`
	CandidateHash string `json:"candidate_hash"`
	State         string `json:"state"`
	Code          string `json:"code"`
	Core          string `json:"core"`
	Connection    string `json:"connection"`
	Tested        int64  `json:"tested"`
	Expires       int64  `json:"expires"`
	ValidUntil    int64  `json:"valid_until"`
	Available     bool   `json:"available"`
}

// Pick one conversion per canonical source identity. A candidate's connection
// evidence belongs only to its own bytes; duplicates never inherit a pass.
const conversionQuery = `WITH choices AS (
 SELECT a.id,m.raw_hash,m.kind,a.expires,j.state,r.candidate_hash,r.code,r.core,r.connection,r.tested,r.valid_until,
 (r.core='passed' AND r.candidate_hash!='' AND r.version=? AND (r.requires_connection=0 OR (r.connection='passed' AND r.valid_until>?))) AS available,
 row_number() OVER(PARTITION BY a.id ORDER BY (r.core='passed' AND r.version=? AND (r.requires_connection=0 OR (r.connection='passed' AND r.valid_until>?))) DESC,(r.connection='passed') DESC,r.tested DESC,m.raw_hash) AS choice
 FROM memberships m JOIN artifacts a ON a.id=m.artifact_id JOIN sources s ON s.id=m.source_id
 LEFT JOIN conversion_jobs j ON j.hash=m.raw_hash LEFT JOIN conversion_results r ON r.hash=m.raw_hash
 WHERE a.expires>? AND s.enabled=1 AND NOT EXISTS(SELECT 1 FROM config_controls c WHERE c.id=a.id AND c.expires>?))
 SELECT id,raw_hash,kind,expires,coalesce(state,'queued'),coalesce(candidate_hash,''),coalesce(code,''),coalesce(core,'not_run'),coalesce(connection,'not_run'),coalesce(tested,0),coalesce(valid_until,0),coalesce(available,0) FROM choices WHERE choice=1`

type conversionQuerier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func (s *Store) Conversions(ctx context.Context, page int, all bool) ([]Conversion, error) {
	return queryConversions(ctx, s.db, page, all)
}

func queryConversions(ctx context.Context, db conversionQuerier, page int, all bool) ([]Conversion, error) {
	now := time.Now().UnixMilli()
	q := conversionQuery + " ORDER BY id"
	args := []any{convert.Version, now, convert.Version, now, now, now}
	if !all {
		q += " LIMIT 100 OFFSET ?"
		args = append(args, max(0, page)*100)
	}
	rows, e := db.QueryContext(ctx, q, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Conversion{}
	for rows.Next() {
		var c Conversion
		if e = rows.Scan(&c.ID, &c.Hash, &c.Kind, &c.Expires, &c.State, &c.CandidateHash, &c.Code, &c.Core, &c.Connection, &c.Tested, &c.ValidUntil, &c.Available); e != nil {
			return nil, e
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
func (s *Store) conversionBytes(c Conversion) ([]byte, error) {
	if !c.Available || !validDigest(c.CandidateHash) {
		return nil, ErrNotFound
	}
	body, e := s.vault.Open(c.CandidateHash)
	if e != nil {
		return nil, e
	}
	defer body.Close()
	b, e := io.ReadAll(io.LimitReader(body, repair.MaxBytes+1))
	if e != nil {
		return nil, e
	}
	var compact bytes.Buffer
	if len(b) > repair.MaxBytes || repair.Digest(b) != c.CandidateHash || json.Compact(&compact, b) != nil || !bytes.HasPrefix(b, []byte("{")) || !bytes.Equal(b, compact.Bytes()) {
		return nil, ErrInvalid
	}
	return b, nil
}

type conversionModule struct{ s *Server }

func (m conversionModule) Register(mux *http.ServeMux) {
	s := m.s
	mux.HandleFunc("GET /api/conversions", func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		entries, e := s.store.Conversions(r.Context(), page, false)
		if e != nil {
			failure(w, e)
			return
		}
		settings, e := s.store.ProbeSettings(r.Context())
		if e != nil {
			failure(w, e)
			return
		}
		worker, e := s.store.ProbeWorker(r.Context())
		if e != nil {
			failure(w, e)
			return
		}
		if time.Now().UnixMilli()-worker.Heartbeat > 30000 {
			worker.Status = "offline"
		}
		respond(w, 200, map[string]any{"entries": entries, "settings": settings, "worker": worker, "export_url": s.origin + "/subscription/" + s.store.digest("subscription", "template-jsonl-v1") + "/configs.jsonl", "single_export_url": s.singleExportURL(), "version": convert.Version, "android_import": "not_run"})
	})
	mux.HandleFunc("POST /api/conversion/settings", func(w http.ResponseWriter, r *http.Request) {
		var v probe.Settings
		if e := decode(r, &v); e != nil {
			failure(w, e)
			return
		}
		if e := s.store.SaveProbeSettings(r.Context(), v); e != nil {
			failure(w, e)
			return
		}
		respond(w, 200, true)
	})
	mux.HandleFunc("GET /output/json/{id}", func(w http.ResponseWriter, r *http.Request) {
		rows, e := s.store.Conversions(r.Context(), 0, true)
		if e != nil {
			failure(w, e)
			return
		}
		for _, c := range rows {
			if c.ID != r.PathValue("id") {
				continue
			}
			b, e := s.store.conversionBytes(c)
			if e != nil {
				_ = s.store.retryConversionBlob(r.Context(), c)
				failure(w, ErrNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("X-Content-SHA256", c.CandidateHash)
			w.Write(b)
			return
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("GET /subscription/{token}/configs.jsonl", s.jsonLines)
}
func (s *Server) jsonLines(w http.ResponseWriter, r *http.Request) {
	if !equal(r.PathValue("token"), s.store.digest("subscription", "template-jsonl-v1")) {
		http.NotFound(w, r)
		return
	}
	rows, e := s.store.Conversions(r.Context(), 0, true)
	if e != nil {
		failure(w, e)
		return
	}
	// Buffer before writing headers, so a failed blob can never corrupt JSONL.
	var out bytes.Buffer
	seen := map[string]bool{}
	count := 0
	for _, c := range rows {
		if !c.Available || seen[c.CandidateHash] {
			continue
		}
		b, e := s.store.conversionBytes(c)
		if e != nil {
			_ = s.store.retryConversionBlob(r.Context(), c)
			continue
		}
		seen[c.CandidateHash] = true
		out.Write(b)
		out.WriteByte('\n')
		count++
	}
	// Keep JSONL bytes intact while letting browsers render the export as text.
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("X-Config-Count", strconv.Itoa(count))
	w.Header().Set("X-Android-Import", "not-run")
	w.Header().Set("X-Verification-Scope", "pinned-core-template; connection-is-separate")
	w.Write(out.Bytes())
}

// This handler is served ONLY on a protected Unix socket, never public TCP.
func (s *Store) PrivateProbeHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /claim", func(w http.ResponseWriter, r *http.Request) {
		j, e := s.ClaimProbe(r.Context())
		if e != nil {
			failure(w, e)
			return
		}
		respond(w, 200, j)
	})
	mux.HandleFunc("POST /renew", func(w http.ResponseWriter, r *http.Request) {
		var j probe.Job
		if e := decode(r, &j); e != nil {
			failure(w, e)
			return
		}
		if e := s.RenewProbe(r.Context(), j); e != nil {
			failure(w, e)
			return
		}
		respond(w, 200, true)
	})
	mux.HandleFunc("POST /complete", func(w http.ResponseWriter, r *http.Request) {
		var v probe.Result
		if e := decode(r, &v); e != nil {
			failure(w, e)
			return
		}
		if e := s.CompleteProbe(r.Context(), v); e != nil {
			failure(w, e)
			return
		}
		respond(w, 200, true)
	})
	mux.HandleFunc("POST /status", func(w http.ResponseWriter, r *http.Request) {
		var v probe.WorkerStatus
		if e := decode(r, &v); e != nil {
			failure(w, e)
			return
		}
		if e := s.SetProbeWorker(r.Context(), v); e != nil {
			failure(w, e)
			return
		}
		respond(w, 200, true)
	})
	return mux
}

// A source/expiry/GC change between list and blob open must not break every
// other exported config. Requeue only the exact still-current broken result;
// a newer worker result must not be invalidated by this stale snapshot.
func (s *Store) retryConversionBlob(ctx context.Context, c Conversion) error {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	res, e := tx.ExecContext(ctx, `UPDATE conversion_results SET candidate_hash='',candidate_size=0,core='blocked',connection='not_run',valid_until=0,code='candidate_blob_unavailable' WHERE hash=? AND candidate_hash=? AND EXISTS(SELECT 1 FROM conversion_inputs i WHERE i.hash=conversion_results.hash)`, c.Hash, c.CandidateHash)
	if e != nil {
		return e
	}
	n, e := res.RowsAffected()
	if e != nil {
		return e
	}
	if n > 0 {
		if _, e = tx.ExecContext(ctx, "UPDATE conversion_jobs SET state='queued',next_run=0,token='',lease_until=0 WHERE hash=?", c.Hash); e != nil {
			return e
		}
	}
	return tx.Commit()
}
