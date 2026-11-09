package collect

import (
	"bytes"
	"context"
	"github.com/rezajafari0970/Config_manager/internal/transient"
	"io"
	"os"
	"testing"
)

func TestAcquisitionSpoolHasNoPlaintextPathAndPreservesDriverReads(t *testing.T) {
	f, path, err := transient.NewFile()
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err = os.Stat(f.Name()); !os.IsNotExist(err) {
		t.Fatal("named plaintext acquisition spool", err)
	}
	raw := []byte("vless://00000000-0000-4000-8000-000000000001@192.0.2.1:443?security=tls#fixture\n")
	if _, err = f.Write(raw); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, raw) {
			t.Fatal("independent offset or bytes lost", err)
		}
		count := 0
		err = NewRegistry().Extract(context.Background(), path, func(_ context.Context, r io.Reader, kind string) error {
			b, e := io.ReadAll(r)
			if kind != "vless" || !bytes.Equal(bytes.TrimSpace(b), bytes.TrimSpace(raw)) {
				t.Error("format plugin lost original bytes")
			}
			count++
			return e
		})
		if err != nil || count != 1 {
			t.Fatal("format plugin could not read anonymous spool", count, err)
		}
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("closed spool still reachable", err)
	}
}
