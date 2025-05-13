package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

var (
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

type CircuitBreaker struct {
	name               string
	failureThreshold   int
	resetTimeout       time.Duration
	halfOpenMaxCalls   int
	state              State
	failures           int
	successes          int
	lastStateChange    time.Time
	mutex              sync.RWMutex
	onStateChange      func(name string, from, to State)
	halfOpenCallsCount int
}

type Option func(*CircuitBreaker)

func WithFailureThreshold(threshold int) Option {
	return func(cb *CircuitBreaker) {
		cb.failureThreshold = threshold
	}
}

func WithResetTimeout(timeout time.Duration) Option {
	return func(cb *CircuitBreaker) {
		cb.resetTimeout = timeout
	}
}

func WithHalfOpenMaxCalls(max int) Option {
	return func(cb *CircuitBreaker) {
		cb.halfOpenMaxCalls = max
	}
}

func WithOnStateChange(callback func(name string, from, to State)) Option {
	return func(cb *CircuitBreaker) {
		cb.onStateChange = callback
	}
}

func NewCircuitBreaker(name string, options ...Option) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:             name,
		failureThreshold: 5,                // Default: 5 failures
		resetTimeout:     10 * time.Second, // Default: 10 seconds
		halfOpenMaxCalls: 1,                // Default: 1 call in half-open state
		state:            StateClosed,
		lastStateChange:  time.Now(),
	}

	for _, option := range options {
		option(cb)
	}

	return cb
}

func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.AllowRequest() {
		return ErrCircuitOpen
	}

	err := fn()

	cb.RecordResult(err == nil)
	return err
}

func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastStateChange) > cb.resetTimeout {
			cb.mutex.RUnlock()
			cb.mutex.Lock()
			defer cb.mutex.Unlock()

			if cb.state == StateOpen && time.Since(cb.lastStateChange) > cb.resetTimeout {
				cb.changeState(StateHalfOpen)
				return true
			}
			return false
		}
		return false
	case StateHalfOpen:
		return cb.halfOpenCallsCount < cb.halfOpenMaxCalls
	default:
		return false
	}
}

func (cb *CircuitBreaker) RecordResult(success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	switch cb.state {
	case StateClosed:
		if !success {
			cb.failures++
			if cb.failures >= cb.failureThreshold {
				cb.changeState(StateOpen)
			}
		} else {
			cb.failures = 0
		}
	case StateHalfOpen:
		cb.halfOpenCallsCount++
		if success {
			cb.successes++
			if cb.successes >= cb.halfOpenMaxCalls {
				cb.changeState(StateClosed)
			}
		} else {
			cb.changeState(StateOpen)
		}
	}
}

func (cb *CircuitBreaker) changeState(newState State) {
	oldState := cb.state
	cb.state = newState
	cb.lastStateChange = time.Now()

	if newState == StateClosed {
		cb.failures = 0
		cb.successes = 0
	} else if newState == StateHalfOpen {
		cb.successes = 0
		cb.halfOpenCallsCount = 0
	}

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, oldState, newState)
	}
}

func (cb *CircuitBreaker) State() State {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) Name() string {
	return cb.name
}
