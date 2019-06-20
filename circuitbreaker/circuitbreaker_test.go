package circuitbreaker

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var testconf = Config{
	TriggerThreshold: 10,
	CountWindow:      time.Second,
	SleepWindow:      500 * time.Millisecond,
	ErrorRate:        0.5,
	HighLatencyValue: time.Second,
	HighLatencyRate:  0.5,
	CoverPanic:       true,
}

func TestCircuitBreakerCloseToOpen(t *testing.T) {
	cb := New(testconf)
	require.Equal(t, cbClose, cb.currentState())

	now := time.Now()
	for i := 0; i < testconf.TriggerThreshold-1; i++ {
		cb.trace(cbClose, true, now)
	}
	require.Equal(t, cbClose, cb.currentState())
	cb.trace(cbClose, false, now)
	require.Equal(t, cbOpen, cb.currentState())
}

func TestCircuitBreakerOpenSleepToHalfOpen(t *testing.T) {
	cb := New(testconf)
	cb.state = cbOpen
	cb.openAt = time.Now().Add(-testconf.SleepWindow + 10*time.Millisecond)
	require.Equal(t, cbOpen, cb.currentState())
	time.Sleep(10 * time.Millisecond)
	require.Equal(t, cbHalfOpen, cb.currentState())
}

func TestCircuitBreakerHalfOpenToOpen(t *testing.T) {
	cb := New(testconf)
	cb.state = cbOpen
	cb.openAt = time.Now().Add(-testconf.SleepWindow + 2*time.Millisecond)
	require.Equal(t, cbOpen, cb.currentState())
	time.Sleep(2 * time.Millisecond)
	require.Equal(t, cbHalfOpen, cb.currentState())
	cb.trace(cbHalfOpen, false, time.Now().Add(-testconf.HighLatencyValue))
	require.Equal(t, cbOpen, cb.currentState())
}

func TestCircuitBreakerHalfOpenToClose(t *testing.T) {
	cb := New(testconf)
	cb.state = cbOpen
	cb.openAt = time.Now().Add(-testconf.SleepWindow + 2*time.Millisecond)
	require.Equal(t, cbOpen, cb.currentState())
	time.Sleep(2 * time.Millisecond)
	require.Equal(t, cbHalfOpen, cb.currentState())
	require.Equal(t, cbOpen, cb.currentState())
	cb.trace(cbHalfOpen, false, time.Now())
	require.Equal(t, cbClose, cb.currentState())
}

func TestCircuitBreaker(t *testing.T) {
	testconf.TriggerThreshold = 20
	testconf.SleepWindow = 100 * time.Millisecond
	cb := New(testconf)

	// OPEN
	wg := sync.WaitGroup{}
	for i := 0; i < testconf.TriggerThreshold; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			err := cb.Run(func() error {
				time.Sleep((testconf.CountWindow / time.Duration(cb.total.size)) * time.Duration(i%4))
				if i%2 == 0 {
					return errors.New("hello")
				}
				return nil
			})
			require.NotEqual(t, ErrIsOpen, err)
		}(i)
	}
	wg.Wait()
	require.True(t, cb.Circuit().IsInterrupted())

	// SLEEP: no request get through
	for i := 0; i < testconf.TriggerThreshold; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			require.True(t, cb.Circuit().IsInterrupted())
		}()
	}
	wg.Wait()
	time.Sleep(testconf.SleepWindow)

	// HALF-OPEN: only one request get through
	var passed int32
	for i := 1; i < testconf.TriggerThreshold; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := cb.Circuit()
			if c.IsInterrupted() {
				return
			}
			c.Trace(true)
			atomic.AddInt32(&passed, 1)
		}()
	}
	wg.Wait()
	require.Equal(t, int32(1), passed)

	{ // OPEN
		c := cb.Circuit()
		require.True(t, c.IsInterrupted())
	}

	for i := 0; i < 3; i++ {
		time.Sleep(testconf.SleepWindow)
		c := cb.Circuit()
		if i == 1 {
			require.True(t, c.IsInterrupted()) // CLOSE -> OPEN
		} else {
			// HALF-OPEN
			require.False(t, c.IsInterrupted())
			c.Trace(false)
		}
	}
	// CLOSE
	for i := 0; i < testconf.TriggerThreshold; i++ {
		c := cb.Circuit()
		require.False(t, c.IsInterrupted())
		c.Trace(false)
	}

	time.Sleep(testconf.CountWindow)
	require.False(t, cb.Circuit().IsInterrupted())
}

func TestCircuitBreakerDsiabled(t *testing.T) {
	testconf.TriggerThreshold = 0
	testconf.HighLatencyValue = 5 * time.Millisecond
	cb := New(testconf)
	for i := 0; i < testconf.TriggerThreshold; i++ {
		err := cb.Run(func() error {
			time.Sleep(testconf.HighLatencyValue)
			return errors.New("world")
		})
		require.NotEqual(t, ErrIsOpen, err)
	}
}
