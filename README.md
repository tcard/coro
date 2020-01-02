# coro [![Build Status](https://secure.travis-ci.org/tcard/coro.svg?branch=master)](http://travis-ci.org/tcard/coro) [![GoDoc](https://godoc.org/github.com/tcard/coro?status.svg)](https://godoc.org/github.com/tcard/coro)

Package coro implements cooperative coroutines on top of goroutines.

It then implements iterators on top of that.

```go
resume := coro.New(func(yield func()) {
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
```

```go
iter := NewStringIterator(func(yield func(string)) error {
	for _, foo := range []string{"foo", "bar", "baz"} {
		yield(foo)
	}
	return errors.New("done")
})

for iter.Next() {
	fmt.Println("yielded:", iter.Yielded)
}
fmt.Println("returned:", iter.Returned)

// Output:
// yielded: foo
// yielded: bar
// yielded: baz
// returned: done
```
