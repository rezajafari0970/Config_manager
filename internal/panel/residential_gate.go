package panel

import (
	"context"
	"errors"
	"net/http"
	"strconv"
)

// Existing export handlers may prepare bytes before a revocation commits.
// Buffer privately on disk, then admit the complete response under the same
// writer boundary as residential revocation. No DB lock spans network output.
type residentialBufferedWriter struct {
	header http.Header
	status int
	out    *exportSpool
	size   int64
	err    error
}

func (w *residentialBufferedWriter) Header() http.Header { return w.header }
func (w *residentialBufferedWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}
func (w *residentialBufferedWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}
	if w.err != nil {
		return 0, w.err
	}
	if w.size+int64(len(b)) > 256<<20 {
		w.err = errors.New("publication_size_limit")
		return 0, w.err
	}
	n, e := w.out.file.Write(b)
	w.size += int64(n)
	w.err = e
	return n, e
}
func publicationStamp(ctx context.Context, q residentialQuery) (string, error) {
	snapshot, e := residentialSnapshot(ctx, q, "all")
	if e != nil {
		return "", e
	}
	var generation int64
	if e = q.QueryRowContext(ctx, "SELECT generation FROM residential_settings WHERE id=1").Scan(&generation); e != nil {
		return "", e
	}
	ads, e := residentialSnapshot(ctx, q, "ads:all")
	if e != nil {
		return "", e
	}
	return snapshot.Generation + ":" + ads.Generation + ":" + strconv.FormatInt(generation, 10), nil
}
func (s *Server) guardLegacyPublication(w http.ResponseWriter, r *http.Request, next http.Handler) {
	before, e := publicationStamp(r.Context(), s.store.reader())
	if e != nil {
		failure(w, e)
		return
	}
	spool, e := newExportSpool()
	if e != nil {
		failure(w, e)
		return
	}
	defer spool.Close()
	buffered := &residentialBufferedWriter{header: w.Header().Clone(), out: spool}
	next.ServeHTTP(buffered, r)
	if buffered.err != nil {
		http.Error(w, "publication preparation failed", 503)
		return
	}
	if buffered.status == 0 {
		buffered.status = 200
	}
	if buffered.status < 300 || buffered.status == http.StatusNotModified {
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
		after, e := publicationStamp(r.Context(), tx)
		if e != nil || after != before {
			http.Error(w, "publication eligibility changed; retry", 503)
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
	}
	if e = spool.Rewind(); e != nil {
		failure(w, e)
		return
	}
	for key := range w.Header() {
		w.Header().Del(key)
	}
	for key, values := range buffered.header {
		w.Header()[key] = values
	}
	w.WriteHeader(buffered.status)
	_, _ = spool.WriteTo(w)
}
