package handlers

import (
	"fmt"
	"time"
)

type waitFn func() (bool, error)

type waiter struct {
	sleep   time.Duration
	timeout time.Duration
	fn      waitFn
}

func Waiter(fn waitFn) *waiter {
	return &waiter{
		fn:      fn,
		sleep:   10 * time.Millisecond, // arbitrary sleep interval
		timeout: 3 * time.Second,       // arbitrary timeout
	}
}

func (w *waiter) Sleep(t time.Duration) *waiter {
	w.sleep = t
	return w
}

func (w *waiter) Timeout(t time.Duration) *waiter {
	w.timeout = t
	return w
}

func (w *waiter) Waitf(format string, args ...interface{}) error {
	var elapsed time.Duration
	for {
		done, err := w.fn()
		if err != nil {
			return err
		} else if done {
			return nil
		}
		time.Sleep(w.sleep)
		elapsed += w.sleep
		if elapsed >= w.timeout {
			return fmt.Errorf(format, args...)
		}
	}
}
