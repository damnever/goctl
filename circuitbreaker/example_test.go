package circuitbreaker

import (
	"fmt"
	"time"
)

func Example() {
	cb := New(DefaultConfig())

	func() (err error) {
		c := cb.Circuit() // DO NOT reuse it, create a new one each time.
		if c.IsInterrupted() {
			err = ErrIsOpen
			return // System is not healthy, return immediately.
		}

		defer func() { c.Trace(err != nil) }()
		// err = doSth()
		fmt.Println(err)
		return
	}()

	func() { // Use Run for "convenience".
		err := cb.Run(func() error {
			time.Sleep(time.Millisecond)
			return fmt.Errorf("is ok")
		})
		fmt.Println(err)
	}()

	// Output:
	// <nil>
	// is ok
}
