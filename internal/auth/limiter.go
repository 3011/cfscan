package auth

import (
	"sync"
	"time"
)

type attemptState struct {
	failures     int
	windowStart  time.Time
	blockedUntil time.Time
}

type LoginLimiter struct {
	mu          sync.Mutex
	attempts    map[string]attemptState
	maxFailures int
	window      time.Duration
	block       time.Duration
	now         func() time.Time
}

func NewLoginLimiter() *LoginLimiter {
	return newLoginLimiter(8, 5*time.Minute, 10*time.Minute, time.Now)
}

func newLoginLimiter(maxFailures int, window, block time.Duration, now func() time.Time) *LoginLimiter {
	return &LoginLimiter{attempts: make(map[string]attemptState), maxFailures: maxFailures, window: window, block: block, now: now}
}

func (l *LoginLimiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	state, ok := l.attempts[key]
	if !ok {
		return true, 0
	}
	if now.Before(state.blockedUntil) {
		return false, state.blockedUntil.Sub(now)
	}
	if now.Sub(state.windowStart) >= l.window {
		delete(l.attempts, key)
	}
	return true, 0
}

func (l *LoginLimiter) Failure(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	state, ok := l.attempts[key]
	if !ok || now.Sub(state.windowStart) >= l.window {
		state = attemptState{windowStart: now}
	}
	state.failures++
	if state.failures >= l.maxFailures {
		state.blockedUntil = now.Add(l.block)
	}
	l.attempts[key] = state
}

func (l *LoginLimiter) Success(key string) {
	l.mu.Lock()
	delete(l.attempts, key)
	l.mu.Unlock()
}
