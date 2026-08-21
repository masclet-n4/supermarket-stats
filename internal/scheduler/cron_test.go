package scheduler

import (
	"testing"
	"time"
)

func TestCronMatches(t *testing.T) {
	c, err := ParseCron("0 3 * * 1-5")
	if err != nil { t.Fatal(err) }
	if !c.Matches(time.Date(2026, 8, 21, 3, 0, 0, 0, time.UTC)) { t.Fatal("expected cron match") }
	if c.Matches(time.Date(2026, 8, 22, 3, 0, 0, 0, time.UTC)) { t.Fatal("unexpected weekend match") }
}
