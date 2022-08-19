package util

import (
	"sync"

	"github.com/relex/gotils/logger"
)

type Pool[T any] struct {
	sync.Pool
}

func NewPool[T any](new func() T) *Pool[T] {
	f := func() any {
		return new()
	}
	return &Pool[T]{
		Pool: sync.Pool{
			New: f,
		},
	}
}

func (pool *Pool[T]) Get() T {
	raw := pool.Pool.Get()
	val, ok := raw.(T)
	if !ok {
		logger.Panic("wrong type of object in Pool: ", raw)
	}
	return val
}

func (pool *Pool[T]) Put(value T) {
	pool.Pool.Put(value)
}
