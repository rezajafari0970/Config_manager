package blob

import (
	"bytes"
	"context"
	"errors"
	"os"
	"sort"
	"testing"
)

func sweepFixture(t *testing.T) (*Vault, []string) {
	t.Helper()
	v, err := New(t.TempDir(), bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	hashes := []string{}
	for i := 0; i < 6; i++ {
		r, err := v.Put(context.Background(), bytes.NewReader([]byte{byte(i)}))
		if err != nil {
			t.Fatal(err)
		}
		v.Release(r.Hash)
		hashes = append(hashes, r.Hash)
	}
	sort.Strings(hashes)
	return v, hashes
}

func TestSweepResumesAfterRepeatedBudgetsAndWraps(t *testing.T) {
	v, hashes := sweepFixture(t)
	seen := map[string]int{}
	for range 3 {
		ctx, cancel := context.WithCancel(context.Background())
		n := 0
		err := v.Sweep(ctx, func(_ context.Context, h string) (bool, error) {
			seen[h]++
			n++
			if n == 2 {
				cancel()
			}
			return true, nil
		})
		cancel()
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	}
	for _, h := range hashes {
		if seen[h] != 1 {
			t.Fatal("tail starved", seen)
		}
	}
	// A new pinned publication survives, including a resumed/wrapped pass.
	newRef, err := v.Put(context.Background(), bytes.NewReader([]byte("new publication")))
	if err != nil {
		t.Fatal(err)
	}
	if err = v.Sweep(context.Background(), func(context.Context, string) (bool, error) { return false, nil }); err != nil {
		t.Fatal(err)
	}
	for _, h := range hashes {
		if _, err = os.Stat(v.path(h)); !os.IsNotExist(err) {
			t.Fatal("unreferenced file remained", err)
		}
	}
	if _, err = os.Stat(v.path(newRef.Hash)); err != nil {
		t.Fatal("pinned file deleted", err)
	}
	v.Release(newRef.Hash)
	if err = v.Sweep(context.Background(), func(context.Context, string) (bool, error) { return false, nil }); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(v.path(newRef.Hash)); !os.IsNotExist(err) {
		t.Fatal(err)
	}
	// Empty directories do not cause a modulo-by-zero on the wrapped traversal.
	if err = v.Sweep(context.Background(), func(context.Context, string) (bool, error) { t.Fatal("empty callback"); return false, nil }); err != nil {
		t.Fatal(err)
	}
}

func TestSweepReferenceErrorPreservesFileWithoutStarvingPeers(t *testing.T) {
	v, hashes := sweepFixture(t)
	failure := errors.New("reference query unavailable")
	check := func(_ context.Context, h string) (bool, error) {
		if h == hashes[0] {
			return false, failure
		}
		return false, nil
	}
	if err := v.Sweep(context.Background(), check); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if err := v.Sweep(context.Background(), check); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if _, err := os.Stat(v.path(hashes[0])); err != nil {
		t.Fatal("query error authorized deletion", err)
	}
	for _, h := range hashes[1:] {
		if _, err := os.Stat(v.path(h)); !os.IsNotExist(err) {
			t.Fatal("error starved later files", err)
		}
	}
	if err := v.Sweep(context.Background(), func(context.Context, string) (bool, error) { return true, nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(v.path(hashes[0])); err != nil {
		t.Fatal(err)
	}
}
