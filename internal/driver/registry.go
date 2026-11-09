package driver

import "sync"

// Registry is the shared compiled-plugin catalog. Readers obtain snapshots;
// plugin callbacks never execute under its lock and may register other plugins.
// Replacing an ID is explicit and does not accumulate obsolete implementations.
// The zero value is ready to use. Plugins themselves must support concurrent use.
type Registry[T any] struct {
	mu     sync.RWMutex
	values map[string]T
	order  []string
}

func (r *Registry[T]) Put(id string, value T) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.values == nil {
		r.values = make(map[string]T)
	}
	if _, ok := r.values[id]; !ok {
		r.order = append(r.order, id)
	}
	r.values[id] = value
}

func (r *Registry[T]) Get(id string) T {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.values[id]
}

func (r *Registry[T]) NewestFirst() []T {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]T, 0, len(r.order))
	for i := len(r.order) - 1; i >= 0; i-- {
		values = append(values, r.values[r.order[i]])
	}
	return values
}
