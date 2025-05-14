package transport

import (
	"encoding/json"
	"net/http"
	"rtcs/internal/circuitbreaker"
	"time"
)

type CircuitBreakerHandler struct {
	registry *circuitbreaker.Registry
}

func NewCircuitBreakerHandler(registry *circuitbreaker.Registry) *CircuitBreakerHandler {
	return &CircuitBreakerHandler{
		registry: registry,
	}
}

type CircuitBreakerStatus struct {
	Name       string `json:"name"`
	State      string `json:"state"`
	LastChange string `json:"last_change"`
}

func (h *CircuitBreakerHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	breakers := h.registry.GetAll()
	statuses := make([]CircuitBreakerStatus, 0, len(breakers))

	for name, cb := range breakers {
		state := "UNKNOWN"
		switch cb.State() {
		case circuitbreaker.StateClosed:
			state = "CLOSED"
		case circuitbreaker.StateOpen:
			state = "OPEN"
		case circuitbreaker.StateHalfOpen:
			state = "HALF-OPEN"
		}

		statuses = append(statuses, CircuitBreakerStatus{
			Name:       name,
			State:      state,
			LastChange: time.Now().Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statuses)
}

func (h *CircuitBreakerHandler) ResetCircuitBreaker(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Missing circuit breaker name", http.StatusBadRequest)
		return
	}

	h.registry.Remove(name)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Circuit breaker reset successfully"))
}
