package pool

import (
	"fmt"
	"time"
)

type SomeResource struct {
	err error
}

func (r *SomeResource) Do() (res string, err error) {
	// if err = doSth(); err != nil {
	//   r.err = err // NOTE: Must record the error if error is not nil.
	//   return
	// }
	res = "hello"
	return
}

func (r *SomeResource) Err() error {
	return r.err
}

func (r *SomeResource) Close() error {
	return nil
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func Example() {
	pool, err := New(Options{
		ResourceFactory: func() (Resource, error) {
			return &SomeResource{}, nil
		},
		Capacity:    5,
		IdleTimeout: 30 * time.Second,
	})
	must(err)
	defer pool.Close()

	r, err := pool.GetNoWait()
	must(err)
	defer pool.Put(r)
	res, err := r.(*SomeResource).Do()
	must(err)
	fmt.Println(res)

	// Output:
	// hello
}
