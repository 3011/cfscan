package scheduling

import (
	"fmt"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/robfig/cron/v3"
)

var parser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

func Next(expression, timezone string, after time.Time) (time.Time, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return time.Time{}, fmt.Errorf("cron_expression is required")
	}
	if timezone == "" {
		timezone = "Asia/Shanghai"
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timezone %q: %w", timezone, err)
	}
	schedule, err := parser.Parse(expression)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid cron expression: %w", err)
	}
	return schedule.Next(after.In(location)).UTC(), nil
}
