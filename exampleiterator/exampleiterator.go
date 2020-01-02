// Package exampleiterator is an example type-safe wrapper of coro.NewIterator.
package exampleiterator

import (
	"github.com/tcard/coro"
)

// Foo is the type that a FooIterator yields.
type Foo string

// NewFooIterator wraps coro.NewIterator with a type-safe interface.
func NewFooIterator(f func(yield func(Foo)) error, options ...coro.SetOption) *FooIterator {
	var it FooIterator
	it.Next = coro.New(func(yield func()) {
		it.Returned = f(func(v Foo) {
			it.Yielded = v
			yield()
		})
	}, options...)
	return &it
}

// A FooIterator holds what's needed to iterate Foos.
type FooIterator struct {
	// Next blocks until the next Foo is set on Yielded, or until the iterator
	// coroutine returns with a (maybe nil) error, which is set on Returned.
	Next     coro.Resume
	Yielded  Foo
	Returned error
}
