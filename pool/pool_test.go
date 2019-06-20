package pool

import (
	"context"
	"errors"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
	require.Nil(t, opts.validate())
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
	require.Nil(t, err)
	resources := []Resource{}
	for i, n := 0, opts.Capacity+1; i < n; i++ {
		r, err := pool.GetNoWait()
		require.True(t, err == nil || err == ErrPoolIsBusy)
		resources = append(resources, r)
	}
	require.Equal(t, opts.Capacity, len(pool.slotsc))
	for _, r := range resources {
		require.Nil(t, pool.Put(r))
	}
	require.Equal(t, opts.Capacity, len(pool.idlec))
	require.Equal(t, pool.IdleNum(), len(pool.idlec))
	require.Equal(t, opts.Capacity, len(pool.slotsc))
	return pool, resources[:opts.Capacity]
}

func TestBasic(t *testing.T) {
	initErr = func() error { return nil }
	opts := Options{ResourceFactory: newFakeResource, Capacity: 3}
	pool, resources := newTestingPool(t, opts)
	defer pool.Close()

	for i := 0; i < opts.Capacity; i++ {
		r, err := pool.GetNoWait()
		require.Nil(t, err)
		exist := false
		for _, rr := range resources {
			if rr == r {
				exist = true
			}
		}
		require.True(t, exist)
	}

	start, done := make(chan struct{}), make(chan struct{})
	go func() {
		<-start
		require.Equal(t, 1, len(pool.idlec))
		require.Equal(t, pool.IdleNum(), len(pool.idlec))
		require.Equal(t, opts.Capacity, len(pool.slotsc))
		r, err := pool.Get(_ctx)
		require.Nil(t, err)
		var _ = r.(*fakeResource)
		require.Equal(t, 0, len(pool.idlec))
		require.Equal(t, opts.Capacity, len(pool.slotsc))
		close(done)
	}()

	_, err := pool.GetNoWait()
	require.Equal(t, ErrPoolIsBusy, err)
	require.Nil(t, pool.Put(resources[0]))
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
	require.Nil(t, err)
	rr0 := r0.(*fakeResetableResource)
	require.Equal(t, 1, rr0.resetcalls)
	require.Nil(t, pool.Put(r0))
	r1, err := pool.GetNoWait()
	require.Nil(t, err)
	rr1 := r1.(*fakeResetableResource)
	require.Equal(t, 2, rr1.resetcalls)
	require.True(t, rr0 == rr1) // We need compare the pointer address!!
	require.Nil(t, pool.Put(r1))

	rr1.reseterr = errors.New("fatal error")
	r2, err := pool.GetNoWait()
	require.Nil(t, err)
	rr2 := r2.(*fakeResetableResource)
	require.Equal(t, 0, rr2.resetcalls)
	require.True(t, rr1 != rr2) // We need compare the pointer address!!
	require.Nil(t, pool.Put(r2))
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
		require.Nil(t, err)
		require.Equal(t, 1, r.(*fakeTestableResource).testcalls)
		require.Nil(t, pool.Put(r))
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
		require.Nil(t, err)
		require.Equal(t, 0, r.(*fakeTestableResource).testcalls)
		require.Nil(t, pool.Put(r))
		time.Sleep(opts.IdleTimeout)
		r1, err := pool.GetNoWait()
		require.Nil(t, err)
		require.True(t, r == r1) // We need compare the pointer address!!
		require.Equal(t, 1, r.(*fakeTestableResource).testcalls)
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
		require.Nil(t, err)
		require.Equal(t, 0, r.(*fakeTestableResource).testcalls)
		require.Nil(t, pool.Put(r))
		time.Sleep(opts.IdleTimeout)
		r1, err := pool.GetNoWait()
		require.Nil(t, err)
		require.True(t, r != r1) // We need compare the pointer address!!
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
	require.Nil(t, err)
	r.(*fakeTestableResource).err = errors.New("fatal")
	require.Nil(t, pool.Put(r))
	require.Equal(t, opts.Capacity-1, len(pool.idlec))
	require.Equal(t, pool.IdleNum(), len(pool.idlec))
	require.Equal(t, opts.Capacity-1, len(pool.slotsc))

	// No new resource created even if pool not full.
	r, err = pool.GetNoWait()
	require.Nil(t, err)
	require.Nil(t, pool.Put(r))
	require.Equal(t, opts.Capacity-1, len(pool.idlec))
	require.Equal(t, opts.Capacity-1, len(pool.slotsc))

	time.Sleep(opts.IdleTimeout)

	// All resources timed out.
	r, err = pool.GetNoWait()
	require.Nil(t, err)
	require.Equal(t, 0, len(pool.idlec))
	require.Equal(t, pool.IdleNum(), len(pool.idlec))
	require.Equal(t, 1, len(pool.slotsc))
	require.Nil(t, pool.Put(r))
	require.Equal(t, 1, len(pool.idlec))
	require.Equal(t, 1, len(pool.slotsc))

	// New resource create by Get.
	r.(*fakeTestableResource).testerr = errors.New("test failed")
	r1, err := pool.Get(_ctx)
	require.Nil(t, err)
	require.NotEqual(t, r, r1)
	require.Equal(t, 0, len(pool.idlec))
	require.Equal(t, pool.IdleNum(), len(pool.idlec))
	require.Equal(t, 1, len(pool.slotsc))

	// Context done.
	resources := []Resource{r1}
	for i, n := 0, opts.Capacity-1; i < n; i++ {
		r, err := pool.GetNoWait()
		require.Nil(t, err)
		resources = append(resources, r)
	}
	ctx, cancel := context.WithCancel(_ctx)
	cancel()
	_, err = pool.Get(ctx)
	require.True(t, err == context.Canceled || err == context.DeadlineExceeded)
	ctx, cancel = context.WithTimeout(_ctx, 10*time.Millisecond)
	defer cancel()
	_, err = pool.Get(ctx)
	require.True(t, err == context.Canceled || err == context.DeadlineExceeded)

	// Closed
	require.Nil(t, pool.Close())
	_, err = pool.GetNoWait()
	require.Equal(t, ErrPoolClosed, err)
	for _, r2 := range resources {
		require.Nil(t, pool.Put(r2))
	}
	require.Equal(t, ErrPoolClosed, pool.Close())
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
	require.Nil(t, err)
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

	require.Equal(t, 0, len(pool.idlec))
	require.Equal(t, pool.IdleNum(), len(pool.idlec))
}
