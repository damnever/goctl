// Package circuitbreaker implements the Circuit-Breaker pattern.
//
// The transformation of circuitbreaker's state is as follows:
//  1. Assuming the number of requests reachs the threshold(Config.TriggerThreshold), initial state is CLOSE.
//  2. After 1, if the error rate or the rate of high latency exceeds the threshold(Config.ErrorRate/HighLatencyRate),
//     the circuitbreaker is OPEN.
//  3. Otherwise for 2, the circuitbreaker will be CLOSE.
//  4. After 2, the circuitbreaker remains OPEN until some amount of time(Config.SleepWindow) pass,
//     then it's the HALF-OPEN which let a single request through, if the request fails, it returns to
//     the OPEN. If the request succeeds, the circuitbreaker is CLOSE, and 1 takes over again.
package circuitbreaker

import (
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"
)

// FIXME(damnever): the two-phase lock may be problematic..

// ErrIsOpen means circuitbreaker is open, the system or API is not healthy.
var ErrIsOpen = errors.New("circuitbreaker is open: not healthy")

type cbState uint8

const (
	cbClose cbState = iota
	cbHalfOpen
	cbOpen
)

// Config configures the CircuitBreaker.
type Config struct {
	// TriggerThreshold is the threshold of total requests to trigger the circuitbreaker,
	// 0 means never trigger.
	TriggerThreshold int
	// CountWindow is the time window to keep stats, any stats before the window will be droped.
	CountWindow time.Duration
	// SleepWindow is the wait time before recovering.
	SleepWindow time.Duration
	// ErrorRate is the error rate to trigger the circuitbreaker.
	ErrorRate float64
	// HighLatencyValue is the max acceptable latency.
	HighLatencyValue time.Duration
	// HighLatencyRate is the rate for HighLatencyValue to trigger the circuitbreaker.
	HighLatencyRate float64
	// CoverPanic will recover the panic, only used in Run.
	CoverPanic bool
}

// DefaultConfig creates a default Config, it is just an example.
func DefaultConfig() Config {
	return Config{
		TriggerThreshold: 10,
		CountWindow:      20 * time.Second,
		SleepWindow:      666 * time.Millisecond,
		ErrorRate:        0.3,
		HighLatencyValue: time.Second,
		HighLatencyRate:  0.5,
		CoverPanic:       true,
	}
}

// CircuitBreaker traces the failures then protects the system.
//
// There is a lock inside, in case of you worry about the performance,
// use multiple CircuitBreaker for defferent kinds of API(request type),
type CircuitBreaker struct {
	disabled bool
	conf     Config

	l        sync.Mutex
	total    *rollingCounter
	errors   *rollingCounter
	timeouts *rollingCounter
	state    cbState
	openAt   time.Time
}

// New creates a new CircuitBreaker. It doesn't check the Config for the caller.
func New(conf Config) *CircuitBreaker {
	if conf.TriggerThreshold <= 0 {
		return &CircuitBreaker{disabled: true}
	}
	now := time.Now()
	return &CircuitBreaker{
		conf:     conf,
		total:    newRollingCounter(conf.CountWindow, now),
		errors:   newRollingCounter(conf.CountWindow, now),
		timeouts: newRollingCounter(conf.CountWindow, now),
	}
}

func (cb *CircuitBreaker) currentState() cbState {
	if cb.disabled {
		return cbClose
	}

	now := time.Now()
	cb.l.Lock()
	defer cb.l.Unlock()

	switch cb.state {
	case cbClose:
		if totalint := cb.total.Count(now); totalint >= cb.conf.TriggerThreshold {
			total := float64(totalint)
			if float64(cb.errors.Count(now))/total >= cb.conf.ErrorRate ||
				float64(cb.timeouts.Count(now))/total >= cb.conf.HighLatencyRate {
				cb.state = cbOpen
				cb.openAt = now
			}
		}
	case cbHalfOpen:
		// Ensure only one request get passed.
		return cbOpen
	case cbOpen:
		if cb.openAt.IsZero() {
			panic("circuitbreaker: corrupted")
		}
		if now.After(cb.openAt.Add(cb.conf.SleepWindow)) {
			cb.state = cbHalfOpen
		}
	}
	return cb.state
}

func (cb *CircuitBreaker) trace(prevstate cbState, haserr bool, start time.Time) {
	if cb.disabled {
		return
	}
	if prevstate == cbOpen {
		panic("circuitbreaker: corrupted")
	}

	isok := true
	now := time.Now()
	cb.l.Lock()
	defer cb.l.Unlock()

	cb.total.Incr(now)
	if haserr {
		cb.errors.Incr(now)
		isok = false
	}
	if now.Sub(start) >= cb.conf.HighLatencyValue {
		cb.timeouts.Incr(now)
		isok = false
	}

	// For the two step locks..
	if prevstate == cbHalfOpen {
		if cb.state != cbHalfOpen {
			panic("circuitbreaker: corrupted")
		}
		if isok {
			cb.state = cbClose
		} else {
			cb.state = cbOpen
			cb.openAt = now
		}
	}
}

// Circuit creates a Circuit, each request(API call) requires exactly one Circuit, DO NOT reuse it or ignore it.
// You can use Run for "convenience".
func (cb *CircuitBreaker) Circuit() Circuit {
	c := Circuit{
		state:   cb.currentState(),
		breaker: cb,
	}
	if c.state != cbOpen {
		c.startat = time.Now()
	}
	return c
}

// Run is a shortcut for the workflow.
func (cb *CircuitBreaker) Run(fn func() error) (err error) {
	state := cb.currentState()
	if state == cbOpen {
		err = ErrIsOpen
		return
	}

	defer func(start time.Time) {
		if cb.conf.CoverPanic {
			if perr := recover(); perr != nil {
				err = fmt.Errorf("%v: %s", perr, debug.Stack())
			}
		}
		cb.trace(state, err != nil, start)
	}(time.Now())

	err = fn()
	return
}

// Circuit is used for every call.
type Circuit struct {
	state   cbState
	startat time.Time
	breaker *CircuitBreaker
}

// IsInterrupted returns true if the circuit is interrupted, the caller should return immediately.
// Otherwise, returns false.
func (c Circuit) IsInterrupted() bool {
	return c.state == cbOpen
}

// Trace records the stats, the caller shouldn't record the "tolerable" error.
// If IsInterrupted returns false, the caller must(use defer) Trace the result,
// otherwise, the circuitbreaker may be OPEN forever.
func (c Circuit) Trace(haserr bool) {
	if c.state == cbOpen {
		return
	}
	c.breaker.trace(c.state, haserr, c.startat)
}
