package cctl

import (
	"context"
	"testing"
	"time"
)

func TestSemaphore(t *testing.T) {
	ctx := context.TODO()
	sem := NewSemaphore(2)
	must(t, sem.Acquire(ctx))
	must(t, sem.Acquire(ctx))

	acquired := make(chan struct{})
	go func() {
		must(t, sem.Acquire(ctx))
		close(acquired)
		must(t, sem.Release())
	}()
	select {
	case <-acquired:
		t.Fatalf("limit exceed")
	case <-time.After(10 * time.Millisecond):
	}
	must(t, sem.Release())
	<-acquired
	must(t, sem.Release())

	if err := sem.Release(); err != ErrOpMismatch {
		t.Fatalf("expect op mismatch")
	}
}

func must(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("require nil, got: %v", err)
	}
}
