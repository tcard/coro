package coro_test

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/tcard/coro/v2"
)

var ctx = context.Background()

func Example() {
	resume := coro.New(ctx, func(yield func()) {
		for i := 1; i <= 3; i++ {
			fmt.Println("coroutine:", i)
			yield()
		}
		fmt.Println("coroutine: done")
	})

	fmt.Println("not started yet")
	for resume() {
		fmt.Println("yielded")
	}
	fmt.Println("returned")

	// Output:
	// not started yet
	// coroutine: 1
	// yielded
	// coroutine: 2
	// yielded
	// coroutine: 3
	// yielded
	// coroutine: done
	// returned
}

func ExampleGenerate() {
	var yielded int
	var err error
	next := coro.Generate(ctx, func(yield func(int)) error {
		for i := 1; i <= 3; i++ {
			yield(i)
		}
		return errors.New("done")
	})

	for next(&err, &yielded) {
		fmt.Println("yielded:", yielded)
	}
	fmt.Println("returned:", err)

	// Output:
	// yielded: 1
	// yielded: 2
	// yielded: 3
	// returned: done
}

func TestLeak(t *testing.T) {
	panicked := make(chan interface{})

	func() {
		resume := coro.NewCoroutine(func(yield func()) {
			defer func() {
				if r := recover(); r != nil {
					panicked <- r
					panic(r)
				}
			}()
			yield()
		})
		resume()
	}()

	for {
		runtime.GC()
		select {
		case p := <-panicked:
			if err, ok := p.(error); !ok || !errors.As(err, &coro.ErrKilled{}) || !errors.Is(err, coro.ErrLeak) {
				t.Errorf("expected ErrLeak within an ErrKilled, got %v", p)
			}
			return
		default:
		}
	}
}

func TestKillOnContextDone(t *testing.T) {
	panicked := make(chan interface{}, 1)

	ctx, cancel := context.WithCancel(context.Background())

	resume := coro.NewCoroutine(func(yield func()) {
		defer func() {
			if r := recover(); r != nil {
				panicked <- r
				panic(r)
			}
		}()

		for {
			yield()
		}
	}, coro.KillOnContextDone(ctx))

	alive := resume()

	select {
	case p := <-panicked:
		t.Fatalf("didn't expect a panic yet, got %v", p)
	default:
	}

	if !alive {
		t.Fatalf("coroutine returned too soon")
	}

	cancel()

	select {
	case p := <-panicked:
		if err, ok := p.(error); !ok || !errors.As(err, &coro.ErrKilled{}) || !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled within an ErrKilled, got %v", p)
		}
	case <-time.After(time.Second):
		t.Fatalf("expected context cancel to cause a panic")
	}

	alive = resume()
	if alive {
		t.Fatalf("coroutine reported as alive on context cancel")
	}
}
