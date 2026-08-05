package plugin

import "sync"

// Registry is a thread-safe, generic plugin registry.
type Registry[K comparable, V any] struct {
	mu    sync.RWMutex
	impls map[K]V
}

func NewRegistry[K comparable, V any]() *Registry[K, V] {
	return &Registry[K, V]{impls: make(map[K]V)}
}

func (r *Registry[K, V]) Register(key K, impl V) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.impls[key] = impl
}

func (r *Registry[K, V]) Get(key K) (V, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.impls[key]
	return v, ok
}

func (r *Registry[K, V]) All() map[K]V {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[K]V, len(r.impls))
	for k, v := range r.impls {
		out[k] = v
	}
	return out
}

func (r *Registry[K, V]) List() []K {
	r.mu.RLock()
	defer r.mu.RUnlock()
	keys := make([]K, 0, len(r.impls))
	for k := range r.impls {
		keys = append(keys, k)
	}
	return keys
}
