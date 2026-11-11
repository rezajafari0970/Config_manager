package panel

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/rezajafari0970/Config_manager/internal/transient"
	"hash"
	"io"
	"os"
)

// Full exports can contain hundreds of large Smart profiles. Spool validated
// lines without retaining the entire subscription in RAM. The private file is
// unlinked immediately: cancellation or process death leaves no plaintext path.
// Headers (including the exact Smart checksum/count) are ready before delivery.
type exportSpool struct {
	file     *os.File
	digest   hash.Hash
	array    bool
	count    int
	finished bool
}

func newExportSpool() (*exportSpool, error) {
	f, _, e := transient.NewFile()
	if e != nil {
		return nil, e
	}
	return &exportSpool{file: f, digest: sha256.New()}, nil
}
func (s *exportSpool) Close() error { return s.file.Close() }
func (s *exportSpool) Line(b []byte) error {
	w := io.MultiWriter(s.file, s.digest)
	if s.array {
		separator := []byte("[")
		if s.count > 0 {
			separator = []byte(",")
		}
		if _, e := w.Write(separator); e != nil {
			return e
		}
		s.count++
	}
	if _, e := w.Write(b); e != nil {
		return e
	}
	_, e := w.Write([]byte{'\n'})
	return e
}
func (s *exportSpool) Hash() string { return hex.EncodeToString(s.digest.Sum(nil)) }
func (s *exportSpool) Rewind() error {
	if s.array && !s.finished {
		ending := "]\n"
		if s.count == 0 {
			ending = "[]\n"
		}
		if _, e := io.MultiWriter(s.file, s.digest).Write([]byte(ending)); e != nil {
			return e
		}
		s.finished = true
	}
	_, e := s.file.Seek(0, io.SeekStart)
	return e
}
func (s *exportSpool) WriteTo(w io.Writer) (int64, error) { return io.Copy(w, s.file) }
