package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

type State int

const (
	Closed State = iota
	Open
	HalfOpen
)

type Breaker struct {
	mu                sync.Mutex
	state             State
	failures          int
	threshold         int
	resetTimeout      time.Duration
	lastFailure       time.Time
	halfOpenMaxCalls  int
	halfOpenCalls     int
}

func New(threshold int, resetTimeout time.Duration) *Breaker {
	return &Breaker{
		state:            Closed,
		threshold:        threshold,
		resetTimeout:     resetTimeout,
		halfOpenMaxCalls: 1,
	}
}

func (b *Breaker) Execute(fn func() error) error {
	b.mu.Lock()
	switch b.state {
	case Open:
		if time.Since(b.lastFailure) > b.resetTimeout {
			b.state = HalfOpen
			b.halfOpenCalls = 0
		} else {
			b.mu.Unlock()
			return ErrCircuitOpen
		}
	case HalfOpen:
		if b.halfOpenCalls >= b.halfOpenMaxCalls {
			b.mu.Unlock()
			return ErrCircuitOpen
		}
		b.halfOpenCalls++
	}
	b.mu.Unlock()

	err := fn()

	b.mu.Lock()
	defer b.mu.Unlock()

	if err != nil {
		b.failures++
		b.lastFailure = time.Now()
		if b.failures >= b.threshold {
			b.state = Open
		}
		return err
	}

	b.failures = 0
	b.state = Closed
	return nil
}

func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}
