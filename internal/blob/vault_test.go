package blob

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestExactStreamingEncryptionAndCleanup(t *testing.T) {
	dir := t.TempDir()
	v, e := New(dir, bytes.Repeat([]byte{4}, 32))
	if e != nil {
		t.Fatal(e)
	}
	raw := append([]byte("  {\r\n \"secret\":\"never-normalize-this\",\"n\":9007199254740993}\r\n"), bytes.Repeat([]byte("x"), 200000)...)
	ref, e := v.Put(context.Background(), bytes.NewReader(raw))
	if e != nil {
		t.Fatal(e)
	}
	stored, _ := os.ReadFile(filepath.Join(dir, ref.Hash+".blob"))
	if bytes.Contains(stored, []byte("never-normalize-this")) {
		t.Fatal("plaintext secret")
	}
	reader, e := v.Open(ref.Hash)
	if e != nil {
		t.Fatal(e)
	}
	got, e := io.ReadAll(reader)
	reader.Close()
	if e != nil || !bytes.Equal(got, raw) {
		t.Fatal("raw bytes changed", e)
	}
	ref2, e := v.Put(context.Background(), bytes.NewReader(raw))
	if e != nil || ref2.Hash != ref.Hash {
		t.Fatal("dedup failed")
	}
	unused := func(context.Context, string) (bool, error) { return false, nil }
	if e = v.Sweep(context.Background(), unused); e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(v.path(ref.Hash)); e != nil {
		t.Fatal("pinned file deleted")
	}
	v.Release(ref.Hash)
	v.Release(ref.Hash)
	if e = v.Sweep(context.Background(), unused); e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(v.path(ref.Hash)); !os.IsNotExist(e) {
		t.Fatal("unreferenced file retained")
	}
}
func TestTamperingAndTruncationFail(t *testing.T) {
	for _, mode := range []string{"tamper", "truncate"} {
		t.Run(mode, func(t *testing.T) {
			v, _ := New(t.TempDir(), bytes.Repeat([]byte{6}, 32))
			ref, e := v.Put(context.Background(), bytes.NewReader([]byte("synthetic-sensitive-config")))
			if e != nil {
				t.Fatal(e)
			}
			path := v.path(ref.Hash)
			data, _ := os.ReadFile(path)
			if mode == "tamper" {
				data[25] ^= 1
			} else {
				data = data[:len(data)-5]
			}
			os.WriteFile(path, data, 0600)
			r, e := v.Open(ref.Hash)
			if e != nil {
				t.Fatal(e)
			}
			_, e = io.ReadAll(r)
			r.Close()
			if e == nil {
				t.Fatal("damaged blob accepted")
			}
		})
	}
}
