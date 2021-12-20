# coro ![Actions](https://github.com/tcard/coro/actions/workflows/go.yml/badge.svg?branch=v2) [![Go Reference](https://pkg.go.dev/badge/github.com/tcard/coro/v2.svg)](https://pkg.go.dev/github.com/tcard/coro/v2)

Package coro implements cooperative coroutines on top of goroutines.

It then implements generators on top of that.

```go
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
```

```go
next := Generate(ctx, func(yield func(string)) error {
	for _, foo := range []string{"foo", "bar", "baz"} {
		yield(foo)
	}
	return errors.New("done")
})

var i int
var err error
for next(&err, &i) {
	fmt.Println("yielded:", i)
}
fmt.Println("returned:", err)

// Output:
// yielded: foo
// yielded: bar
// yielded: baz
// returned: done
```
