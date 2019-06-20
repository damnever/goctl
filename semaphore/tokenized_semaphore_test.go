package semaphore

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTokenizedSemaphore1(t *testing.T) {
	sem := NewTokenizedSemaphore(1)
	tokens := strings.Split("a b c d e f g", " ")
	testTokenizedSemaphore1(t, sem, tokens)
}

func TestTokenizedSemaphore2(t *testing.T) {
	sem := NewTokenizedSemaphore(2)
	tokens := strings.Split("a a a a a a a", " ")
	testTokenizedSemaphore1(t, sem, tokens)
}

func testTokenizedSemaphore1(t *testing.T, sem *TokenizedSemaphore, tokens []string) {
	ctx := context.TODO()
	doneC := make(chan string, len(tokens))
	for _, token := range tokens {
		go func(token string) {
			require.Nil(t, sem.Acquire(ctx, token))
			doneC <- token
		}(token)
	}
	for i, n := 0, len(tokens); i < n; i++ {
		token := <-doneC
		select {
		case <-doneC:
			t.Fatal("limit exceed")
		case <-time.After(100 * time.Millisecond):
		}
		require.Nil(t, sem.Release(token))
	}
}

func TestTokenizedSemaphore3(t *testing.T) {
	ctx := context.TODO()
	sem := NewTokenizedSemaphore(2)

	type control struct {
		acquireC     chan struct{}
		releaseC     chan struct{}
		acquireDoneC chan struct{}
		releaseDoneC chan struct{}
	}
	newControl := func() *control {
		return &control{
			acquireC:     make(chan struct{}),
			releaseC:     make(chan struct{}),
			acquireDoneC: make(chan struct{}),
			releaseDoneC: make(chan struct{}),
		}
	}

	ctl1 := newControl()
	go func() {
		<-ctl1.acquireC
		require.Nil(t, sem.Acquire(ctx, "a"))
		close(ctl1.acquireDoneC)
		<-ctl1.releaseC
		require.Nil(t, sem.Release("a"))
		close(ctl1.releaseDoneC)
	}()
	ctl2 := newControl()
	go func() {
		<-ctl2.acquireC
		time.Sleep(5 * time.Millisecond)
		require.Nil(t, sem.Acquire(ctx, "a"))
		close(ctl2.acquireDoneC)
		<-ctl2.releaseC
		require.Nil(t, sem.Release("a"))
		close(ctl2.releaseDoneC)
	}()
	ctl3 := newControl()
	go func() {
		<-ctl3.acquireC
		time.Sleep(10 * time.Millisecond)
		require.Nil(t, sem.Acquire(ctx, "b"))
		close(ctl3.acquireDoneC)
		<-ctl3.releaseC
		require.Nil(t, sem.Release("b"))
		close(ctl3.releaseDoneC)
	}()

	close(ctl1.acquireC)
	close(ctl2.acquireC)
	close(ctl3.acquireC)
	for i, c := range []chan struct{}{
		ctl1.acquireDoneC,
		ctl3.acquireDoneC,
	} {
		select {
		case <-c:
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("%d not done", i)
		}
	}
	select {
	case <-ctl2.acquireDoneC:
		t.Fatalf("limit exceed")
	case <-time.After(100 * time.Millisecond):
	}
	close(ctl1.releaseC)
	close(ctl2.releaseC)
	close(ctl3.releaseC)

	donexx := make(chan struct{})
	go func() {
		require.Nil(t, sem.Acquire(ctx, "c"))
		require.Nil(t, sem.Release("c"))
		close(donexx)
	}()

	for i, c := range []chan struct{}{
		ctl1.releaseDoneC,
		ctl2.releaseDoneC,
		ctl3.releaseDoneC,
		donexx,
	} {
		select {
		case <-c:
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("%d not done", i)
		}
	}
}
