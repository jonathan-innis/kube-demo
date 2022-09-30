package atomic

import "sync"

type MapIter[K comparable, V any] func(K, V)

type Map[K comparable, V any] struct {
	mu  *sync.Mutex
	raw map[K]V
}

func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{
		mu:  &sync.Mutex{},
		raw: make(map[K]V),
	}
}

func (m *Map[K, V]) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.raw)
}

func (m *Map[K, V]) Load(k K, v V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.raw[k] = v
}

func (m *Map[K, V]) Get(k K) (V, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.raw[k]; ok {
		return m.raw[k], true
	}
	return *new(V), false
}

func (m *Map[K, V]) Range(iter MapIter[K, V]) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, v := range m.raw {
		iter(k, v)
	}
}

func (m *Map[K, V]) Delete(k K) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.raw, k)
}
