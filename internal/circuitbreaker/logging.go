package circuitbreaker

import (
	"log"
)

func LoggingMiddleware(cb *CircuitBreaker) *CircuitBreaker {
	originalCallback := cb.onStateChange

	cb.onStateChange = func(name string, from, to State) {
		log.Printf("[CIRCUIT BREAKER] %s changed from %s to %s", name, stateToString(from), stateToString(to))

		if originalCallback != nil {
			originalCallback(name, from, to)
		}
	}

	return cb
}

func stateToString(state State) string {
	switch state {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF-OPEN"
	default:
		return "UNKNOWN"
	}
}

func MetricsMiddleware(cb *CircuitBreaker) *CircuitBreaker {
	originalCallback := cb.onStateChange

	cb.onStateChange = func(name string, from, to State) {

		if originalCallback != nil {
			originalCallback(name, from, to)
		}
	}

	return cb
}
