package panel

import (
	"bytes"
	"github.com/rezajafari0970/Config_manager/internal/repair"
	"io"
	"os"
	"testing"
)

func TestLargeExportSpoolExactBytesChecksumAndNoPlaintextPath(t *testing.T) {
	s, e := newExportSpool()
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if _, e = os.Stat(s.file.Name()); !os.IsNotExist(e) {
		t.Fatal("export leaves named plaintext file")
	}
	line := append([]byte(`{"fixture":"`), bytes.Repeat([]byte("x"), 64*1024)...)
	line = append(line, []byte(`"}`)...)
	var want bytes.Buffer
	for i := 0; i < 150; i++ {
		if e = s.Line(line); e != nil {
			t.Fatal(e)
		}
		want.Write(line)
		want.WriteByte('\n')
	}
	if s.Hash() != repair.Digest(want.Bytes()) {
		t.Fatal("export checksum mismatch")
	}
	if e = s.Rewind(); e != nil {
		t.Fatal(e)
	}
	got, e := io.ReadAll(s.file)
	if e != nil || !bytes.Equal(got, want.Bytes()) {
		t.Fatal("JSONL bytes changed", e)
	}
	s.Close()
	if e = s.Line(line); e == nil {
		t.Fatal("storage failure was ignored")
	}
}
