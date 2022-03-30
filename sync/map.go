package sync

import "sync"

type Map[K, V any] struct {
	sync.Map
}

func (m *Map[K, V]) Delete(v K) {
	m.Map.Delete(v)
}

func (m *Map[K, V]) Load(key K) (value V, ok bool) {
	vInterface, ok := m.Map.Load(key)
	if !ok {
		var zeroValue V
		return zeroValue, false
	}
	return vInterface.(V), true
}

func (m *Map[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	vInterface, ok := m.Map.LoadAndDelete(key)
	if !ok {
		var zeroValue V
		return zeroValue, false
	}
	return vInterface.(V), true
}

func (m *Map[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	vInterface, loaded := m.Map.LoadOrStore(key, value)
	return vInterface.(V), loaded
}

func (m *Map[K, V]) Range(f func(key K, value V) bool) {
	m.Map.Range(func(kInterface interface{}, vInterface interface{}) bool {
		return f(kInterface.(K), vInterface.(V))
	})
}

func (m *Map[K, V]) Store(key K, value V) {
	m.Map.Store(key, value)
}
