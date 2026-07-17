package scheduling

import (
	"testing"
	"time"
)

func TestNextUsesRequestedTimezone(t *testing.T) {
	after := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	next, err := Next("0 8 * * *", "Asia/Shanghai", after)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("got %s want %s", next, want)
	}
}

func TestNextRejectsInvalidCron(t *testing.T) {
	if _, err := Next("* * *", "UTC", time.Now()); err == nil {
		t.Fatal("expected invalid cron error")
	}
}
