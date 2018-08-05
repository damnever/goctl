package circuitbreaker

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
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
	assert(t, cb.currentState(), cbClose)

	now := time.Now()
	for i := 0; i < testconf.TriggerThreshold-1; i++ {
		cb.trace(cbClose, true, now)
	}
	assert(t, cb.currentState(), cbClose)
	cb.trace(cbClose, false, now)
	assert(t, cb.currentState(), cbOpen)
}

func TestCircuitBreakerOpenSleepToHalfOpen(t *testing.T) {
	cb := New(testconf)
	cb.state = cbOpen
	cb.openAt = time.Now().Add(-testconf.SleepWindow + 10*time.Millisecond)
	assert(t, cb.currentState(), cbOpen)
	time.Sleep(10 * time.Millisecond)
	assert(t, cb.currentState(), cbHalfOpen)
}

func TestCircuitBreakerHalfOpenToOpen(t *testing.T) {
	cb := New(testconf)
	cb.state = cbOpen
	cb.openAt = time.Now().Add(-testconf.SleepWindow + 2*time.Millisecond)
	assert(t, cb.currentState(), cbOpen)
	time.Sleep(2 * time.Millisecond)
	assert(t, cb.currentState(), cbHalfOpen)
	cb.trace(cbHalfOpen, false, time.Now().Add(-testconf.HighLatencyValue))
	assert(t, cb.currentState(), cbOpen)
}

func TestCircuitBreakerHalfOpenToClose(t *testing.T) {
	cb := New(testconf)
	cb.state = cbOpen
	cb.openAt = time.Now().Add(-testconf.SleepWindow + 2*time.Millisecond)
	assert(t, cb.currentState(), cbOpen)
	time.Sleep(2 * time.Millisecond)
	assert(t, cb.currentState(), cbHalfOpen)
	assert(t, cb.currentState(), cbOpen)
	cb.trace(cbHalfOpen, false, time.Now())
	assert(t, cb.currentState(), cbClose)
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
			assert(t, err != ErrIsOpen, true)
		}(i)
	}
	wg.Wait()
	assert(t, cb.Circuit().IsInterrupted(), true)

	// SLEEP: no request get through
	for i := 0; i < testconf.TriggerThreshold; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			assert(t, cb.Circuit().IsInterrupted(), true)
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
	assert(t, passed, int32(1))

	{ // OPEN
		c := cb.Circuit()
		assert(t, c.IsInterrupted(), true)
	}

	for i := 0; i < 3; i++ {
		time.Sleep(testconf.SleepWindow)
		c := cb.Circuit()
		if i == 1 {
			assert(t, c.IsInterrupted(), true) // CLOSE -> OPEN
		} else {
			// HALF-OPEN
			assert(t, c.IsInterrupted(), false)
			c.Trace(false)
		}
	}
	// CLOSE
	for i := 0; i < testconf.TriggerThreshold; i++ {
		c := cb.Circuit()
		assert(t, c.IsInterrupted(), false)
		c.Trace(false)
	}

	time.Sleep(testconf.CountWindow)
	assert(t, cb.Circuit().IsInterrupted(), false)
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
		assert(t, err != ErrIsOpen, true)
	}
}
