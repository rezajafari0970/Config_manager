package blob

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
)

type acquisitionErrorReader struct {
	n    int
	done bool
}

func (r *acquisitionErrorReader) Read(p []byte) (int, error) {
	if r.done {
		return 0, io.ErrUnexpectedEOF
	}
	r.done = true
	n := min(r.n, len(p))
	copy(p, bytes.Repeat([]byte("x"), n))
	return n, io.ErrUnexpectedEOF
}
func TestUnderlyingTruncationNeverBecomesAuthenticatedBlob(t *testing.T) {
	for _, n := range []int{7, chunkSize} {
		v, _ := New(t.TempDir(), bytes.Repeat([]byte{1}, 32))
		if _, err := v.Put(context.Background(), &acquisitionErrorReader{n: n}); err != io.ErrUnexpectedEOF {
			t.Fatal("underlying error hidden", n, err)
		}
		entries, _ := os.ReadDir(v.dir)
		if len(entries) != 0 {
			t.Fatal("partial blob persisted")
		}
	}
}
func TestAuthenticatedDuplicateKeepsImmutableCiphertext(t *testing.T) {
	v, _ := New(t.TempDir(), bytes.Repeat([]byte{2}, 32))
	raw := []byte("exact synthetic bytes\r\n")
	a, err := v.Put(context.Background(), bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(v.path(a.Hash))
	b, err := v.Put(context.Background(), bytes.NewReader(raw))
	if err != nil || a != b {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(v.path(a.Hash))
	if !bytes.Equal(before, after) {
		t.Fatal("healthy duplicate rewritten")
	}
	v.Release(a.Hash)
	v.Release(b.Hash)
	if err = v.Sweep(context.Background(), func(context.Context, string) (bool, error) { return false, nil }); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(v.path(a.Hash)); !os.IsNotExist(err) {
		t.Fatal("duplicate pin leaked")
	}
}

func TestEmptyAndChunkBoundaryCiphertextCannotReadAsSuccess(t *testing.T) {
	for _, boundary := range []int{0, 16, 16 + 4 + chunkSize + 16} {
		v, _ := New(t.TempDir(), bytes.Repeat([]byte{3}, 32))
		ref, err := v.Put(context.Background(), bytes.NewReader(bytes.Repeat([]byte("x"), chunkSize+30)))
		if err != nil {
			t.Fatal(err)
		}
		path := v.path(ref.Hash)
		if err = os.Truncate(path, int64(boundary)); err != nil {
			t.Fatal(err)
		}
		r, err := v.Open(ref.Hash)
		if err != nil {
			t.Fatal(err)
		}
		_, err = io.ReadAll(r)
		r.Close()
		if err == nil {
			t.Fatalf("unauthenticated EOF accepted at %d", boundary)
		}
	}
}
