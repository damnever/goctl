package pool

import (
	"context"
	"errors"
	"math/rand"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestOptions(t *testing.T) {
	opts := Options{}
	if err := opts.validate(); err == nil || !strings.Contains(err.Error(), "ResourceFactory") {
		t.Fatalf("expect error about ResourceFactory, got: %v", err)
	}
	opts.ResourceFactory = func() (Resource, error) { return nil, nil }
	if err := opts.validate(); err == nil || !strings.Contains(err.Error(), "Capacity") {
		t.Fatalf("expect error about Capacity, got: %v", err)
	}
	opts.Capacity = 3
	assertNil(t, opts.validate())
}

var (
	initErr = func() error { return nil }
	_ctx    = context.TODO()
)

type fakeResource struct {
	err      error
	closeErr error
}

func newFakeResource() (Resource, error) {
	return &fakeResource{}, initErr()
}

func (r *fakeResource) Err() error {
	return r.err
}

func (r *fakeResource) Close() error {
	return r.closeErr
}

type fakeResetableResource struct {
	*fakeResource
	resetcalls int
	reseterr   error
}

func newFakeResetableResource() (Resource, error) {
	return &fakeResetableResource{fakeResource: &fakeResource{}}, initErr()
}

func (rr *fakeResetableResource) Reset() error {
	rr.resetcalls++
	return rr.reseterr
}

type fakeTestableResource struct {
	*fakeResource
	testcalls int
	testerr   error
}

func newFakeTestableResource() (Resource, error) {
	return &fakeTestableResource{fakeResource: &fakeResource{}}, initErr()
}

func (tr *fakeTestableResource) Test() error {
	tr.testcalls++
	return tr.testerr
}

func newTestingPool(t *testing.T, opts Options) (*Pool, []Resource) {
	pool, err := New(opts)
	assertNil(t, err)
	resources := []Resource{}
	for i, n := 0, opts.Capacity+1; i < n; i++ {
		r, err := pool.GetNoWait()
		assertTrue(t, err == nil || err == ErrPoolIsBusy)
		resources = append(resources, r)
	}
	assertTrue(t, len(pool.slotsc) == opts.Capacity)
	for _, r := range resources {
		assertNil(t, pool.Put(r))
	}
	assertTrue(t, len(pool.idlec) == opts.Capacity)
	assertTrue(t, len(pool.idlec) == pool.IdleNum())
	assertTrue(t, len(pool.slotsc) == opts.Capacity)
	return pool, resources[:opts.Capacity]
}

func TestBasic(t *testing.T) {
	initErr = func() error { return nil }
	opts := Options{ResourceFactory: newFakeResource, Capacity: 3}
	pool, resources := newTestingPool(t, opts)
	defer pool.Close()

	for i := 0; i < opts.Capacity; i++ {
		r, err := pool.GetNoWait()
		assertNil(t, err)
		exist := false
		for _, rr := range resources {
			if rr == r {
				exist = true
			}
		}
		assertTrue(t, exist)
	}

	start, done := make(chan struct{}), make(chan struct{})
	go func() {
		<-start
		assertTrue(t, len(pool.idlec) == 1)
		assertTrue(t, len(pool.idlec) == pool.IdleNum())
		assertTrue(t, len(pool.slotsc) == opts.Capacity)
		r, err := pool.Get(_ctx)
		assertNil(t, err)
		var _ = r.(*fakeResource)
		assertTrue(t, len(pool.idlec) == 0)
		assertTrue(t, len(pool.slotsc) == opts.Capacity)
		close(done)
	}()

	_, err := pool.GetNoWait()
	assertTrue(t, err == ErrPoolIsBusy)
	assertNil(t, pool.Put(resources[0]))
	close(start)
	select {
	case <-done:
	case <-time.After(time.Millisecond * 10):
		t.Fatal("timedout")
	}
}

func TestResourceReset(t *testing.T) {
	initErr = func() error { return nil }
	pool, _ := newTestingPool(t, Options{
		ResourceFactory: newFakeResetableResource,
		Capacity:        1,
		ResetOnBorrow:   true,
	})
	defer pool.Close()
	r0, err := pool.GetNoWait()
	assertNil(t, err)
	rr0 := r0.(*fakeResetableResource)
	assertTrue(t, rr0.resetcalls == 1)
	assertNil(t, pool.Put(r0))
	r1, err := pool.GetNoWait()
	assertNil(t, err)
	rr1 := r1.(*fakeResetableResource)
	assertTrue(t, rr1.resetcalls == 2)
	assertTrue(t, rr0 == rr1)
	assertNil(t, pool.Put(r1))

	rr1.reseterr = errors.New("fatal error")
	r2, err := pool.GetNoWait()
	assertNil(t, err)
	rr2 := r2.(*fakeResetableResource)
	assertTrue(t, rr2.resetcalls == 0)
	assertTrue(t, rr1 != rr2)
	assertNil(t, pool.Put(r2))
}

func TestResourceTesting(t *testing.T) {
	initErr = func() error { return nil }
	{
		opts := Options{
			ResourceFactory: newFakeTestableResource,
			Capacity:        1,
			TestOnBorrow:    true,
		}
		pool, _ := newTestingPool(t, opts)
		defer pool.Close()
		r, err := pool.GetNoWait()
		assertNil(t, err)
		assertTrue(t, r.(*fakeTestableResource).testcalls == 1)
		assertNil(t, pool.Put(r))
	}
	{
		opts := Options{
			ResourceFactory: newFakeTestableResource,
			Capacity:        1,
			IdleTimeout:     20 * time.Millisecond,
			TestWhileIdle:   true,
		}
		pool, _ := newTestingPool(t, opts)
		defer pool.Close()

		r, err := pool.GetNoWait()
		assertNil(t, err)
		assertTrue(t, r.(*fakeTestableResource).testcalls == 0)
		assertNil(t, pool.Put(r))
		time.Sleep(opts.IdleTimeout)
		r1, err := pool.GetNoWait()
		assertNil(t, err)
		assertTrue(t, r == r1)
		assertTrue(t, r.(*fakeTestableResource).testcalls == 1)
	}
	{
		opts := Options{
			ResourceFactory: newFakeTestableResource,
			Capacity:        1,
			IdleTimeout:     20 * time.Millisecond,
		}
		pool, _ := newTestingPool(t, opts)
		defer pool.Close()

		r, err := pool.GetNoWait()
		assertNil(t, err)
		assertTrue(t, r.(*fakeTestableResource).testcalls == 0)
		assertNil(t, pool.Put(r))
		time.Sleep(opts.IdleTimeout)
		r1, err := pool.GetNoWait()
		assertNil(t, err)
		assertTrue(t, r != r1)
	}
}

func TestErrors(t *testing.T) {
	opts := Options{
		ResourceFactory: newFakeTestableResource,
		Capacity:        3,
		IdleTimeout:     100 * time.Millisecond,
		TestOnBorrow:    true,
	}
	pool, _ := newTestingPool(t, opts)

	// Put with error.
	r, err := pool.GetNoWait()
	assertNil(t, err)
	r.(*fakeTestableResource).err = errors.New("fatal")
	assertNil(t, pool.Put(r))
	assertTrue(t, len(pool.idlec) == opts.Capacity-1)
	assertTrue(t, len(pool.idlec) == pool.IdleNum())
	assertTrue(t, len(pool.slotsc) == opts.Capacity-1)

	// No new resource created even if pool not full.
	r, err = pool.GetNoWait()
	assertNil(t, err)
	assertNil(t, pool.Put(r))
	assertTrue(t, len(pool.idlec) == opts.Capacity-1)
	assertTrue(t, len(pool.slotsc) == opts.Capacity-1)

	time.Sleep(opts.IdleTimeout)

	// All resources timed out.
	r, err = pool.GetNoWait()
	assertNil(t, err)
	assertTrue(t, len(pool.idlec) == 0)
	assertTrue(t, len(pool.idlec) == pool.IdleNum())
	assertTrue(t, len(pool.slotsc) == 1)
	assertNil(t, pool.Put(r))
	assertTrue(t, len(pool.idlec) == 1)
	assertTrue(t, len(pool.slotsc) == 1)

	// New resource create by Get.
	r.(*fakeTestableResource).testerr = errors.New("test failed")
	r1, err := pool.Get(_ctx)
	assertNil(t, err)
	assertTrue(t, r != r1)
	assertTrue(t, len(pool.idlec) == 0)
	assertTrue(t, len(pool.idlec) == pool.IdleNum())
	assertTrue(t, len(pool.slotsc) == 1)

	// Context done.
	resources := []Resource{r1}
	for i, n := 0, opts.Capacity-1; i < n; i++ {
		r, err := pool.GetNoWait()
		assertNil(t, err)
		resources = append(resources, r)
	}
	ctx, cancel := context.WithCancel(_ctx)
	cancel()
	_, err = pool.Get(ctx)
	assertTrue(t, err == context.Canceled || err == context.DeadlineExceeded)
	ctx, cancel = context.WithTimeout(_ctx, 10*time.Millisecond)
	defer cancel()
	_, err = pool.Get(ctx)
	assertTrue(t, err == context.Canceled || err == context.DeadlineExceeded)

	// Closed
	assertNil(t, pool.Close())
	_, err = pool.GetNoWait()
	assertTrue(t, err == ErrPoolClosed)
	for _, r2 := range resources {
		assertNil(t, pool.Put(r2))
	}
	assertTrue(t, pool.Close() == ErrPoolClosed)
}

func TestConcurrentOps(t *testing.T) {
	initErr = func() error {
		if rand.Intn(10) > 8 {
			return errors.New("init error")
		}
		return nil
	}
	opts := Options{
		ResourceFactory: newFakeTestableResource,
		Capacity:        20,
		IdleTimeout:     10 * time.Millisecond,
		TestOnBorrow:    true,
	}
	pool, err := New(opts)
	assertNil(t, err)
	defer pool.Close()

	wg := sync.WaitGroup{}
	for i, n := 0, opts.Capacity*300; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(time.Millisecond * time.Duration(20*rand.Float64()))
			r, err := pool.Get(_ctx)
			if err != nil {
				return
			}
			raw := r.(*fakeTestableResource)
			if rn := rand.Intn(10); rn <= 2 {
				time.Sleep(opts.IdleTimeout / 2)
			} else if rn <= 4 {
				time.Sleep(opts.IdleTimeout)
			} else if rn <= 6 {
				raw.testerr = errors.New("fatal error")
			} else if rn <= 8 {
				raw.err = errors.New("fatal error")
			}
			pool.Put(r)
		}()

		if i == n-101 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				time.Sleep(opts.IdleTimeout)
				pool.Close()
			}()
		}
	}
	wg.Wait()

	assertTrue(t, len(pool.idlec) == 0)
	assertTrue(t, len(pool.idlec) == pool.IdleNum())
}

func assertTrue(t *testing.T, v bool) {
	if !v {
		_, _, line, _ := runtime.Caller(1)
		t.Fatalf("[%d] expect true, got false", line)
	}
}

func assertNil(t *testing.T, v interface{}) {
	if v != nil {
		_, _, line, _ := runtime.Caller(1)
		t.Fatalf("[%d] expect nil, got: %v", line, v)
	}
}
