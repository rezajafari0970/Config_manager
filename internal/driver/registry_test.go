package driver

import (
	"fmt"
	"sync"
	"testing"
)

func TestRegistryConcurrentSnapshotsAndReplacement(t *testing.T) {
	var r Registry[int]
	r.Put("first", 1)
	r.Put("second", 2)
	snapshot := r.NewestFirst()
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for n := 0; n < 200; n++ {
				r.Put(fmt.Sprint(g), n)
				_ = r.Get("first")
				_ = r.NewestFirst()
			}
		}(g)
	}
	wg.Wait()
	r.Put("first", 3)
	if r.Get("first") != 3 || len(r.NewestFirst()) != 10 || snapshot[0] != 2 || snapshot[1] != 1 {
		t.Fatal("replacement mutated snapshots or duplicated plugin")
	}
}
