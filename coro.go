// Package coro implements cooperative coroutines on top of goroutines.
//
// coro implements a concurrency model similar to Lua's coroutines, Python's
// generators or Ruby's fibers, in which several concurrent processes exist
// at the same time, but whose executions don't overlap; instead, they
// explicitly yield control to each other.
//
// The coroutine protocol
//
// coro provides a base protocol for coordinating goroutines. In this protocol,
// there is a goroutine whose execution only proceeds when other goroutines
// decide so; in turn, those goroutines block until the former goroutine either
// finishes or "yields".
//
// Such goroutine, which we call "coroutine", is created with the New function.
// To New, you pass a function that defines the coroutine's execution much like
// you pass a function to the 'go' statement that defines the goroutine's
// execution. The difference is that the coroutine doesn't start right away;
// instead, another goroutine must call the Resume function returned by New.
//
// The Resume function blocks the calling goroutine while the coroutine is
// executing. The coroutine may then call the 'yield' function which is passed
// to its defining function. 'yield', in turn, blocks the coroutine until the
// Resume function is called again.
//
// Thus, while a goroutine is blocked on calling a Resume func, the coroutine is
// executing; and, while the coroutine is blocked on calling its 'yield' func,
// the other goroutine is executing.
//
// Resume is also called when the coroutine's defining function returns, in
// which case it returns false.
//
// Since the participating goroutine's executions never overlap and have a
// well-defined order, they are synchronized.
//
// Killing and cancelling coroutines
//
// To help prevent goroutine leaks, when a coroutine is blocked on a 'yield' and
// the library detects that no other goroutine will ever resume it, the call to
// 'yield' will panic with an ErrKilled error wrapping an ErrLeak.
//
// Additionally, you can tie the coroutine's lifetime to a context by passing
// the KillOnContextDone option. When the context is cancelled or reaches its
// deadline, the coroutine is killed.
//
// This kind of panic is recovered by the library. The coroutine's function may
// intercept such panics in its own deferred recovery code.
//
// The killed coroutine's Resume func, if ever called, will return false, as if
// the coroutine had exited normally.
//
// Behavior on panics
//
// If the coroutine's goroutine panics, its Resume func returns false, as if the
// coroutine had exited normally.
package coro

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
)

// Resume is an alias for a function that yields control to a coroutine,
// blocking until the coroutine either yields control back or returns.
type Resume = func() (alive bool)

// Options is an internal configuration type. It's configured via SetOptions
// provided when creating a coroutine with New.
type Options struct {
	g       GoFunc
	killCtx context.Context
}

// A SetOption sets an option on the
type SetOption func(*Options)

// KillOnContextDone configures a coroutine to be killed when the provided
// context is done.
func KillOnContextDone(ctx context.Context) SetOption {
	return func(o *Options) {
		o.killCtx = ctx
	}
}

// A GoFunc spawns goroutines.
type GoFunc func(func())

// WithGoFunc sets a custom GoFunc to spawn goroutines.
func WithGoFunc(g GoFunc) SetOption {
	return func(o *Options) {
		o.g = g
	}
}

var defaultOptions = []SetOption{
	KillOnContextDone(context.Background()),
	WithGoFunc(func(f func()) { go f() }),
}

// New creates a coroutine.
//
// See package-level documentation for details on how coroutines work.
func New(f func(yield func()), setOptions ...SetOption) Resume {
	var options Options
	for _, setOption := range append(defaultOptions, setOptions...) {
		setOption(&options)
	}

	yieldCh := make(chan struct{})
	garbageCollected := make(chan struct{})

	var resumeToken bool
	resume := func() bool {
		resumeToken = !resumeToken

		// resume...
		_, ok := <-yieldCh
		if !ok {
			// resumed dead coroutine
			return false
		}

		// ... and wait for yield or return
		_, ok = <-yieldCh
		return ok
	}

	runtime.SetFinalizer(&resumeToken, func(interface{}) {
		close(garbageCollected)
	})

	var yieldPanic error

	waitResume := func() {
		select {
		case yieldCh <- struct{}{}:
			return
		case <-garbageCollected:
			yieldPanic = ErrKilled{ErrLeak}
		case <-options.killCtx.Done():
			yieldPanic = ErrKilled{options.killCtx.Err()}
		}
		panic(yieldPanic)
	}

	options.g(func() {
		defer close(yieldCh)

		defer func() {
			r := recover()
			if r == nil {
				return
			}
			if err, ok := r.(error); ok && errors.As(err, &ErrKilled{}) {
				return
			}
			panic(r)
		}()

		waitResume()

		f(func() {
			if yieldPanic != nil {
				panic(yieldPanic)
			}

			// make call to Resume return
			yieldCh <- struct{}{}

			waitResume()
		})
	})

	return resume
}

// ErrLeak is the error with which a coroutine is killed when it's
// detected to be stuck forever.
//
// Currently, this means that the coroutine's associated Resume function has
// been garbage-collected.
var ErrLeak = errors.New("coro: coroutine leaked")

// An ErrKilled is the error with which the library kills a goroutine.
//
// See package-level documentation for details.
type ErrKilled struct {
	By error
}

func (err ErrKilled) Error() string {
	return fmt.Errorf("coro: coroutine killed: %w", err.By).Error()
}

func (err ErrKilled) Unwrap() error {
	return err.By
}

// NewIterator implements an iterator protocol on top of a raw coroutine.
//
// When the coroutine yields, it calls a yield function with a value. This
// value is set on the 'yielded' parameter, which must be a pointer to a
// value settable to the yielded value.
//
// When the coroutine returns, the return value is set in the same way on the
// 'returned' parameter.
//
// See package exampleiterator for an example of a type-safe wrapper to this
// function.
func NewIterator(yielded, returned interface{}, f func(yield func(interface{})) interface{}, setOption ...SetOption) Resume {
	setYielded := reflect.ValueOf(yielded).Elem().Set
	setReturned := reflect.ValueOf(returned).Elem().Set
	return New(func(yield func()) {
		returned := f(func(v interface{}) {
			setYielded(reflect.ValueOf(v))
			yield()
		})
		setReturned(reflect.ValueOf(returned))
	}, setOption...)
}
