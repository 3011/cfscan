package enrollment

import (
	"testing"
	"time"
)

func TestRateLimiterResetsAfterWindow(t *testing.T) {
	now := time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)
	limiter := newRateLimiter(2, time.Minute, func() time.Time { return now })
	if allowed, _ := limiter.Allow("client"); !allowed {
		t.Fatal("first request should pass")
	}
	if allowed, _ := limiter.Allow("client"); !allowed {
		t.Fatal("second request should pass")
	}
	if allowed, retry := limiter.Allow("client"); allowed || retry != time.Minute {
		t.Fatalf("expected rate limit, allowed=%v retry=%s", allowed, retry)
	}
	now = now.Add(time.Minute)
	if allowed, _ := limiter.Allow("client"); !allowed {
		t.Fatal("request should pass after reset")
	}
}
