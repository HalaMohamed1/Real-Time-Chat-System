package circuitbreaker

import (
	"sync"
)

type Registry struct {
	breakers map[string]*CircuitBreaker
	mutex    sync.RWMutex
}

func NewRegistry() *Registry {
	return &Registry{
		breakers: make(map[string]*CircuitBreaker),
	}
}

func (r *Registry) Get(name string, options ...Option) *CircuitBreaker {
	r.mutex.RLock()
	cb, exists := r.breakers[name]
	r.mutex.RUnlock()

	if exists {
		return cb
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if cb, exists = r.breakers[name]; exists {
		return cb
	}

	cb = NewCircuitBreaker(name, options...)
	r.breakers[name] = cb
	return cb
}

func (r *Registry) Remove(name string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.breakers, name)
}

func (r *Registry) GetAll() map[string]*CircuitBreaker {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make(map[string]*CircuitBreaker, len(r.breakers))
	for k, v := range r.breakers {
		result[k] = v
	}
	return result
}
