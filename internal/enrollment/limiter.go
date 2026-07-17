package enrollment

import (
	"sync"
	"time"
)

type rateState struct {
	count       int
	windowStart time.Time
}

type RateLimiter struct {
	mu     sync.Mutex
	states map[string]rateState
	limit  int
	window time.Duration
	now    func() time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return newRateLimiter(limit, window, time.Now)
}

func newRateLimiter(limit int, window time.Duration, now func() time.Time) *RateLimiter {
	return &RateLimiter{states: make(map[string]rateState), limit: limit, window: window, now: now}
}

func (l *RateLimiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	state, ok := l.states[key]
	if !ok || now.Sub(state.windowStart) >= l.window {
		l.states[key] = rateState{count: 1, windowStart: now}
		return true, 0
	}
	if state.count >= l.limit {
		return false, max(l.window-now.Sub(state.windowStart), time.Second)
	}
	state.count++
	l.states[key] = state
	return true, 0
}
