package pool

import "testing"

func BenchmarkGetPut(b *testing.B) {
	opts := Options{
		ResourceFactory: newFakeTestableResource,
		Capacity:        20,
	}
	pool, _ := New(opts)
	defer pool.Close()

	for i := 0; i < b.N; i++ {
		for i := 0; i < opts.Capacity; i++ {
			r, _ := pool.GetNoWait()
			pool.Put(r)
		}
	}
}
