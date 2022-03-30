package sync

import "sync"

func NewPool[T any]() Pool[T] {
	return Pool[T]{
		&sync.Pool{
			New: func() interface{} {
				return new(T)
			},
		},
	}
}

type Pool[T any] struct {
	*sync.Pool
}

func (mp Pool[T]) Get() *T {
	return mp.Pool.Get().(*T)
}

func (mp Pool[T]) Put(t *T) {
	mp.Pool.Put(t)
}
