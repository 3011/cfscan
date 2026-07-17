package auth

import (
	"testing"
	"time"
)

func TestLoginLimiterBlocksAndResets(t *testing.T) {
	now := time.Date(2026, 7, 16, 8, 0, 0, 0, time.UTC)
	limiter := newLoginLimiter(3, time.Minute, 2*time.Minute, func() time.Time { return now })
	for i := 0; i < 3; i++ {
		if allowed, _ := limiter.Allow("client"); !allowed {
			t.Fatalf("attempt %d should still be allowed before recording failure", i+1)
		}
		limiter.Failure("client")
	}
	if allowed, retry := limiter.Allow("client"); allowed || retry != 2*time.Minute {
		t.Fatalf("expected blocked client with two minute retry, got allowed=%v retry=%s", allowed, retry)
	}
	now = now.Add(2 * time.Minute)
	if allowed, _ := limiter.Allow("client"); !allowed {
		t.Fatal("expected block to expire")
	}
	limiter.Failure("client")
	limiter.Success("client")
	if allowed, _ := limiter.Allow("client"); !allowed {
		t.Fatal("expected successful login to clear failures")
	}
}
