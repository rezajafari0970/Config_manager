package panel

import (
	"context"
	"database/sql"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"github.com/rezajafari0970/Config_manager/internal/residential"
	"net/http"
	"time"
)

func (s *Store) registerPrivateResidential(mux *http.ServeMux) {
	mux.HandleFunc("POST /residential/claim", func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Worker string `json:"worker"`
		}
		if decode(r, &request) != nil || request.Worker != "" && !safeCode(request.Worker) {
			failure(w, ErrInvalid)
			return
		}
		j, e := s.claimResidentialFor(r.Context(), request.Worker)
		if e != nil {
			failure(w, e)
			return
		}
		respond(w, 200, j)
	})
	mux.HandleFunc("POST /residential/renew", func(w http.ResponseWriter, r *http.Request) {
		var j residential.Job
		if e := decode(r, &j); e != nil {
			failure(w, e)
			return
		}
		if e := s.RenewResidential(r.Context(), j); e != nil {
			failure(w, e)
			return
		}
		respond(w, 200, true)
	})
	mux.HandleFunc("POST /residential/complete", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 6<<20)
		var c residential.Completion
		if e := decode(r, &c); e != nil {
			failure(w, e)
			return
		}
		if e := s.CompleteResidential(r.Context(), c); e != nil {
			failure(w, e)
			return
		}
		respond(w, 200, true)
	})
	mux.HandleFunc("POST /residential/status", func(w http.ResponseWriter, r *http.Request) {
		var v probe.WorkerStatus
		if e := decode(r, &v); e != nil {
			failure(w, e)
			return
		}
		if v.Version != residential.Version || v.Status != "running" && v.Status != "starting" && v.Status != "blocked" && v.Status != "offline" || v.Code != "" && !safeCode(v.Code) || v.Active < 0 || v.Active > 8 {
			failure(w, ErrInvalid)
			return
		}
		_, e := s.db.ExecContext(r.Context(), "UPDATE residential_worker SET status=?,code=?,version=?,active=?,heartbeat=? WHERE id=1", v.Status, v.Code, v.Version, v.Active, time.Now().UnixMilli())
		if e != nil {
			failure(w, e)
			return
		}
		respond(w, 200, true)
	})
}
func (s *Store) manualResidentialDelete(ctx context.Context, tx *sql.Tx, all bool, now int64) error {
	query := `SELECT DISTINCT r.hash FROM residential_records r JOIN residential_owners o ON o.hash=r.hash WHERE r.state!='DELETED' AND o.identity IN (SELECT id FROM selected_configs)`
	if all {
		query = `SELECT hash FROM residential_records WHERE state!='DELETED'`
	}
	rows, e := tx.QueryContext(ctx, query)
	if e != nil {
		return e
	}
	var hashes []string
	for rows.Next() {
		var hash string
		if e = rows.Scan(&hash); e != nil {
			rows.Close()
			return e
		}
		hashes = append(hashes, hash)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, hash := range hashes {
		var accepted bool
		if e = tx.QueryRowContext(ctx, "SELECT ever_accepted FROM residential_records WHERE hash=?", hash).Scan(&accepted); e != nil {
			return e
		}
		if accepted {
			if e = s.revokeResidential(ctx, tx, hash, "manual_delete", true, now); e != nil {
				return e
			}
		} else {
			// Preserve existing ordinary deletion/explicit source-restore semantics for
			// inputs that have NEVER been accepted residential. Fence their pending job.
			if _, e = tx.ExecContext(ctx, "UPDATE residential_records SET state='DELETED',reason='manual_delete',candidate_hash='',candidate_size=0,report='{}',decision='{}',revision=revision+1,epoch=epoch+1,token='',lease_until=0,valid_until=0 WHERE hash=?", hash); e != nil {
				return e
			}
		}
	}
	return nil
}

func (s *Server) residentialManage(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Hash   string `json:"hash"`
		Action string `json:"action"`
	}
	if decode(r, &request) != nil || !validDigest(request.Hash) || (request.Action != "delete" && request.Action != "recheck" && request.Action != "disable") {
		failure(w, ErrInvalid)
		return
	}
	tx, e := s.store.db.BeginTx(r.Context(), nil)
	if e != nil {
		failure(w, e)
		return
	}
	defer tx.Rollback()
	var state string
	if e = tx.QueryRowContext(r.Context(), "SELECT state FROM residential_records WHERE hash=?", request.Hash).Scan(&state); e != nil {
		failure(w, ErrNotFound)
		return
	}
	if state == "DELETED" {
		failure(w, ErrConflict)
		return
	}
	if request.Action == "delete" {
		e = s.store.revokeResidential(r.Context(), tx, request.Hash, "manual_delete", true, time.Now().UnixMilli())
	} else {
		_, e = tx.ExecContext(r.Context(), "UPDATE residential_records SET paused=?,state=CASE WHEN ever_accepted=1 THEN 'QUARANTINED' ELSE 'UNKNOWN' END,valid_until=0,reason='manual_recheck',revision=revision+1,token='',lease_until=0,next_run=0 WHERE hash=?", request.Action == "disable", request.Hash)
	}
	if e != nil {
		failure(w, e)
		return
	}
	if e = tx.Commit(); e != nil {
		failure(w, e)
		return
	}
	respond(w, 200, true)
}
