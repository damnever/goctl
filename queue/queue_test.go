package queue

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestQueueGetNoWait(t *testing.T) {
	q := NewQueue()
	for i := 0; i < 10; i++ {
		q.Put(i)
	}
	for i := 0; i < 10; i++ {
		item, err := q.GetNoWait()
		require.Nil(t, err)
		require.Equal(t, i, item.(int))
	}
	_, err := q.GetNoWait()
	require.Equal(t, ErrEmpty, err)
}

func TestQeueGet(t *testing.T) {
	q := NewQueue()

	valc := make(chan interface{})
	go func() {
		item, err := q.Get(context.TODO())
		if err != nil {
			valc <- err
		} else {
			valc <- item
		}
	}()

	time.Sleep(20 * time.Millisecond)
	select {
	case v := <-valc:
		t.Fatalf("except nothing, got: %v", v)
	default:
	}
	q.Put(1)
	select {
	case v := <-valc:
		_, ok := v.(int)
		require.Equal(t, true, ok)
	case <-time.After(30 * time.Millisecond):
		t.Fatalf("Get timed out")
	}

	ctx, cancel := context.WithCancel(context.TODO())
	go func() {
		_, err := q.Get(ctx)
		valc <- err
	}()
	cancel()
	select {
	case v := <-valc:
		require.Equal(t, context.Canceled, v)
	case <-time.After(30 * time.Millisecond):
		t.Fatalf("Op timed out")
	}
}

func TestQueueFIFO(t *testing.T) {
	q := NewQueue()
	donec := make(chan struct{})
	go func() {
		for i := 0; i < 2; i++ {
			item, err := q.Get(context.TODO())
			require.Nil(t, err)
			require.Equal(t, i, item.(int))
		}
		close(donec)
	}()
	time.Sleep(20 * time.Millisecond) // Wait until Get block
	q.l.Lock()
	q.items.Append(0)
	q.l.Unlock()
	q.Put(1)
	select {
	case <-donec:
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("Op timed out")
	}
}

func TestQueueConcurrentOPS(t *testing.T) {
	q := NewQueue()
	N := 10000
	wg := sync.WaitGroup{}

	for i := 0; i < N; i += 10 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			for j := i + 10; i < j; i++ {
				if sleep := r.Intn(99); sleep%2 != 0 {
					time.Sleep(time.Duration(sleep) * time.Millisecond)
				}
				q.Put(i)
			}
		}(i)
	}

	var count int32
	for i := 0; i < N; i += 5 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			for j := i + 5; i < j; i++ {
				if sleep := r.Intn(11) + 1; sleep%2 == 0 {
					time.Sleep(time.Duration(sleep) * time.Millisecond)
				}
				item, err := q.Get(context.TODO())
				require.Nil(t, err)
				atomic.AddInt32(&count, int32(item.(int)))
			}
		}(i)
	}

	wg.Wait()
	require.Equal(t, N*(N-1)/2, int(count))
}
