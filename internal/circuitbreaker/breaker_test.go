package circuitbreaker

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBreaker_StartsClosed(t *testing.T) {
	b := New(3, time.Second)
	assert.Equal(t, Closed, b.State())
}

func TestBreaker_OpensAfterThreshold(t *testing.T) {
	b := New(3, time.Second)

	testErr := errors.New("fail")
	for i := 0; i < 3; i++ {
		err := b.Execute(func() error { return testErr })
		assert.Error(t, err)
	}

	assert.Equal(t, Open, b.State())

	err := b.Execute(func() error { return nil })
	assert.ErrorIs(t, err, ErrCircuitOpen)
}

func TestBreaker_ClosesOnSuccess(t *testing.T) {
	b := New(3, time.Second)

	// 2 failures (below threshold)
	for i := 0; i < 2; i++ {
		b.Execute(func() error { return errors.New("fail") })
	}

	// Success resets counter
	err := b.Execute(func() error { return nil })
	assert.NoError(t, err)
	assert.Equal(t, Closed, b.State())
}

func TestBreaker_HalfOpenAfterTimeout(t *testing.T) {
	b := New(2, 50*time.Millisecond)

	// Open the breaker
	b.Execute(func() error { return errors.New("fail") })
	b.Execute(func() error { return errors.New("fail") })
	assert.Equal(t, Open, b.State())

	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)

	// Half-open: one call allowed
	err := b.Execute(func() error { return nil })
	assert.NoError(t, err)
	assert.Equal(t, Closed, b.State())
}

func TestBreaker_HalfOpen_FailsAgain(t *testing.T) {
	b := New(2, 50*time.Millisecond)

	b.Execute(func() error { return errors.New("fail") })
	b.Execute(func() error { return errors.New("fail") })
	assert.Equal(t, Open, b.State())

	time.Sleep(60 * time.Millisecond)

	// Half-open: call fails -> back to open
	err := b.Execute(func() error { return errors.New("fail") })
	assert.Error(t, err)
	assert.Equal(t, Open, b.State())
}
