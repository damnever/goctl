// Package retry provides util functions to retry fail actions.
package retry

import (
	"errors"
	"time"
)

// ErrNeedRetry is a placholder helper, in case you have no error to return, such as bool status, etc.
var ErrNeedRetry = errors.New("need retry")

// State controls whether the fail action should continue retrying.
type State uint8

const (
	// Continue continues retrying the fail action.
	Continue State = iota
	// StopWithErr stops retrying the fail action,
	// returns the error which the RetryFunc returns.
	StopWithErr
	// StopWithNil stops retrying the fail action, returns nil.
	StopWithNil
)

// Retrier retrys fail actions with backoff.
type Retrier struct {
	backoffs []time.Duration
}

// New creates a new Retrier with backoffs, the backoffs is the wait
// time before each retrying.
// The count of retrying will be len(backoffs), the first call
// is not counted in retrying.
func New(backoffs []time.Duration) Retrier {
	return Retrier{backoffs: append(backoffs, 0)}
}

// Run keeps calling the RetryFunc if it returns (Continue, non-nil-err),
// otherwise it will stop retrying. It is goroutine safe unless you do something wrong ^_^.
func (r Retrier) Run(try func() (State, error)) (err error) {
	var state State
	for _, backoff := range r.backoffs {
		state, err = try()
		switch state {
		case StopWithErr:
			return err
		case StopWithNil:
			return nil
		default: // Continue
		}
		if err == nil {
			return nil
		}
		if backoff > 0 {
			time.Sleep(backoff)
		}
	}
	return err
}

// Retry is a shortcut for Retrier.Run.
func Retry(backoffs []time.Duration, try func() (State, error)) error {
	return New(backoffs).Run(try)
}

// ConstantBackoffs creates a list of backoffs with constant values.
func ConstantBackoffs(n int, backoff time.Duration) []time.Duration {
	backoffs := make([]time.Duration, n, n+1)
	if backoff > 0 {
		for i := 0; i < n; i++ {
			backoffs[i] = backoff
		}
	}
	return backoffs
}

// ZeroBackoffs creates a list of backoffs with zero values.
func ZeroBackoffs(n int) []time.Duration {
	return ConstantBackoffs(n, 0)
}

// ExponentialBackoffs creates a list of backoffs with values are calculated by backoff*2^[0 1 2 .. n).
func ExponentialBackoffs(n int, backoff time.Duration) []time.Duration {
	backoffs := make([]time.Duration, n, n+1)
	if backoff > 0 {
		for i := 0; i < n; i++ {
			backoffs[i] = backoff * (1 << uint(i))
		}
	}
	return backoffs
}
