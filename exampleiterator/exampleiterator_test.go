package exampleiterator

import (
	"errors"
	"fmt"
)

func Example() {
	iter := NewFooIterator(func(yield func(Foo)) error {
		for _, foo := range []Foo{"foo", "bar", "baz"} {
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
}
