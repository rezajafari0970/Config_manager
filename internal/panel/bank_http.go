package panel

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/collect"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"github.com/rezajafari0970/Config_manager/internal/repair"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type bankModule struct{ s *Server }

func (m bankModule) Register(mux *http.ServeMux) {
	s := m.s
	mux.HandleFunc("GET /api/bank", s.bankState)
	mux.HandleFunc("GET /api/bank/report/{id}", s.bankReport)
	mux.HandleFunc("GET /bank/variant/{id}/{hash}", s.bankVariant)
	mux.HandleFunc("POST /api/bank/import", s.bankImport)
	mux.HandleFunc("POST /api/bank/batch", func(w http.ResponseWriter, r *http.Request) {
		var b ConfigBatch
		if e := decode(r, &b); e != nil {
			failure(w, e)
			return
		}
		n, e := s.store.ManageBank(r.Context(), b)
		if e != nil {
			failure(w, e)
			return
		}
		respond(w, 200, map[string]int64{"changed": n})
	})
	mux.HandleFunc("POST /api/probe/settings", func(w http.ResponseWriter, r *http.Request) {
		var v probe.Settings
		if e := decode(r, &v); e != nil {
			failure(w, e)
			return
		}
		if e := s.store.SaveProbeSettings(r.Context(), v); e != nil {
			failure(w, e)
			return
		}
		respond(w, 200, map[string]bool{"saved": true})
	})
	mux.HandleFunc("GET /bank/raw/{id}", func(w http.ResponseWriter, r *http.Request) {
		v, e := s.store.BankOne(r.Context(), r.PathValue("id"))
		if e != nil {
			failure(w, e)
			return
		}
		s.stream(w, r, v.Hash, v.Size, v.Hash[:12]+".txt")
	})
	mux.HandleFunc("GET /bank/candidate/{id}", func(w http.ResponseWriter, r *http.Request) {
		v, e := s.store.BankOne(r.Context(), r.PathValue("id"))
		if e != nil {
			failure(w, e)
			return
		}
		if !v.Enabled || v.CandidateHash == "" {
			failure(w, ErrNotFound)
			return
		}
		var size int64
		if e = s.store.db.QueryRowContext(r.Context(), "SELECT candidate_size FROM probe_results WHERE hash=?", v.Hash).Scan(&size); e != nil {
			failure(w, e)
			return
		}
		w.Header().Set("X-Android-Import", "not-run")
		s.stream(w, r, v.CandidateHash, size, v.Hash[:12]+".json")
	})
	mux.HandleFunc("GET /sub/{token}/{group}", s.subscription)
}
func (s *Server) bankState(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	ctx := r.Context()
	entries, e := s.store.Bank(ctx, r.URL.Query().Get("kind"), r.URL.Query().Get("state"), page)
	if e != nil {
		failure(w, e)
		return
	}
	stats, e := s.store.BankStats(ctx)
	if e != nil {
		failure(w, e)
		return
	}
	settings, e := s.store.ProbeSettings(ctx)
	if e != nil {
		failure(w, e)
		return
	}
	worker, e := s.store.ProbeWorker(ctx)
	if e != nil {
		failure(w, e)
		return
	}
	if time.Now().UnixMilli()-worker.Heartbeat > 30000 {
		worker.Status = "offline"
	}
	var models []map[string]any
	rows, e := s.store.db.QueryContext(ctx, "SELECT model,kind,observed,passed,failed,diagnosis FROM probe_models ORDER BY last_seen DESC LIMIT 20")
	if e != nil {
		failure(w, e)
		return
	}
	for rows.Next() {
		var model, kind, code string
		var observed, passed, failed int64
		if e = rows.Scan(&model, &kind, &observed, &passed, &failed, &code); e != nil {
			break
		}
		models = append(models, map[string]any{"model": model, "kind": kind, "attempts": observed, "passed": passed, "failed": failed, "code": code})
	}
	rows.Close()
	if e != nil {
		failure(w, e)
		return
	}
	base := s.origin + "/sub/" + s.store.digest("subscription", "server-verified-v1") + "/"
	respond(w, 200, map[string]any{"entries": entries, "stats": stats, "settings": settings, "worker": worker, "models": models, "now": time.Now().UnixMilli(), "subscriptions": map[string]string{"all": base + "all.json", "repaired": base + "repaired.json"}})
}
func (s *Store) ImportBank(ctx context.Context, body io.Reader) (int, error) {
	f, e := os.CreateTemp("", "config-bank-import-")
	if e != nil {
		return 0, e
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if _, e = io.Copy(f, body); e != nil {
		return 0, e
	}
	count := 0
	e = collect.NewRegistry().Extract(ctx, f.Name(), func(ctx context.Context, r io.Reader, kind string) error {
		ref, e := s.vault.Put(ctx, r)
		if e != nil {
			return e
		}
		defer s.vault.Release(ref.Hash)
		stream, e := s.vault.Open(ref.Hash)
		if e != nil {
			return e
		}
		sample, e := io.ReadAll(io.LimitReader(stream, repair.MaxBytes+1))
		stream.Close()
		if e != nil {
			return e
		}
		identity := collect.Fingerprint(sample, int64(len(sample)) == ref.Size, kind, ref.Hash)
		tx, e := s.db.BeginTx(ctx, nil)
		if e != nil {
			return e
		}
		defer tx.Rollback()
		if e = saveSample(ctx, tx, ref.Hash, identity, kind, ref.Size, time.Now().UnixMilli(), "ورودی دستی"); e != nil {
			return e
		}
		if e = tx.Commit(); e == nil {
			count++
		}
		return e
	})
	return count, e
}
func (s *Server) bankImport(w http.ResponseWriter, r *http.Request) {
	n, e := s.store.ImportBank(r.Context(), r.Body)
	if e != nil {
		failure(w, e)
		return
	}
	respond(w, 200, map[string]int{"received": n})
}

// The proof gate is queried at export time. Source expiry, stale cached UI state,
// failed retests and disabled records cannot bypass it.
func (s *Store) Subscription(ctx context.Context, repaired bool) ([]json.RawMessage, error) {
	q := `SELECT b.identity,r.candidate_hash FROM bank_samples b JOIN probe_results r ON r.hash=b.hash WHERE b.deleted=0 AND b.enabled=1 AND r.core='passed' AND r.connection='passed' AND r.valid_until>? AND r.version=? AND r.candidate_hash!=''`
	if repaired {
		q += ` AND r.rule NOT IN ('original-json',?)`
	}
	args := []any{time.Now().UnixMilli(), probe.Version}
	if repaired {
		args = append(args, probe.Version+".uri-json")
	}
	q += " ORDER BY r.tested DESC,b.hash"
	rows, e := s.db.QueryContext(ctx, q, args...)
	if e != nil {
		return nil, e
	}
	type ref struct{ id, hash string }
	refs := []ref{}
	for rows.Next() {
		var v ref
		if e = rows.Scan(&v.id, &v.hash); e != nil {
			break
		}
		refs = append(refs, v)
	}
	if e == nil {
		e = rows.Err()
	}
	rows.Close()
	if e != nil {
		return nil, e
	}
	seenID := map[string]bool{}
	seenHash := map[string]bool{}
	out := []json.RawMessage{}
	for _, v := range refs {
		if seenID[v.id] || seenHash[v.hash] {
			continue
		}
		body, e := s.vault.Open(v.hash)
		if e != nil {
			continue
		}
		b, e := io.ReadAll(io.LimitReader(body, repair.MaxBytes+1))
		body.Close()
		if e != nil || repair.Digest(b) != v.hash || !json.Valid(b) || !bytes.HasPrefix(bytes.TrimSpace(b), []byte("{")) {
			continue
		}
		seenID[v.id] = true
		seenHash[v.hash] = true
		out = append(out, json.RawMessage(b))
	}
	return out, nil
}
func (s *Server) subscription(w http.ResponseWriter, r *http.Request) {
	if !equal(r.PathValue("token"), s.store.digest("subscription", "server-verified-v1")) || r.PathValue("group") != "all.json" && r.PathValue("group") != "repaired.json" {
		http.NotFound(w, r)
		return
	}
	configs, e := s.store.Subscription(r.Context(), r.PathValue("group") == "repaired.json")
	if e != nil {
		failure(w, e)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Android-Import", "not-run")
	w.Header().Set("X-Verification-Scope", "pinned-linux-core-and-two-https-destinations")
	w.Header().Set("Profile-Update-Interval", "1")
	w.Header().Set("X-Config-Count", strconv.Itoa(len(configs)))
	// Preserve each candidate's exact tested JSON bytes inside the array envelope.
	w.Write([]byte("[\n"))
	for i, b := range configs {
		if i > 0 {
			w.Write([]byte(",\n"))
		}
		w.Write(b)
	}
	w.Write([]byte("\n]\n"))
}

// PrivateHandler MUST only be mounted on the protected Unix socket, never TCP.
func (s *Store) PrivateProbeHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /strategy", func(w http.ResponseWriter, r *http.Request) {
		var v struct {
			Model string `json:"model"`
		}
		if e := decode(r, &v); e != nil {
			failure(w, e)
			return
		}
		rule, e := s.PreferredRule(r.Context(), v.Model)
		if e != nil {
			failure(w, e)
			return
		}
		respond(w, 200, rule)
	})
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
