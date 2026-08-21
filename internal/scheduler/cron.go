package scheduler

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Cron struct{ fields [5]map[int]bool }

func ParseCron(expression string) (Cron, error) {
	parts := strings.Fields(expression)
	if len(parts) != 5 { return Cron{}, fmt.Errorf("cron must have 5 fields: %q", expression) }
	ranges := [5][2]int{{0, 59}, {0, 23}, {1, 31}, {1, 12}, {0, 6}}
	var c Cron
	for i, part := range parts {
		values, err := parseField(part, ranges[i][0], ranges[i][1])
		if err != nil { return Cron{}, fmt.Errorf("field %d: %w", i, err) }
		c.fields[i] = values
	}
	return c, nil
}

func (c Cron) String() string { return "5-field cron" }

func (c Cron) Matches(t time.Time) bool {
	return c.fields[0][t.Minute()] && c.fields[1][t.Hour()] && c.fields[2][t.Day()] && c.fields[3][int(t.Month())] && c.fields[4][int(t.Weekday())]
}

func (c Cron) Next(after time.Time) time.Time {
	for t := after.Truncate(time.Minute).Add(time.Minute); ; t = t.Add(time.Minute) {
		if c.Matches(t) { return t }
	}
}

func parseField(value string, min, max int) (map[int]bool, error) {
	result := map[int]bool{}
	for _, item := range strings.Split(value, ",") {
		base, step := item, 1
		if strings.Contains(item, "/") {
			parts := strings.Split(item, "/")
			if len(parts) != 2 { return nil, fmt.Errorf("invalid step %q", item) }
			base = parts[0]
			var err error
			step, err = strconv.Atoi(parts[1])
			if err != nil || step < 1 { return nil, fmt.Errorf("invalid step %q", item) }
		}
		start, end := min, max
		if base != "*" {
			if strings.Contains(base, "-") {
				parts := strings.Split(base, "-")
				if len(parts) != 2 { return nil, fmt.Errorf("invalid range %q", base) }
				var err error
				start, err = strconv.Atoi(parts[0]); if err != nil { return nil, err }
				end, err = strconv.Atoi(parts[1]); if err != nil { return nil, err }
			} else {
				var err error
				start, err = strconv.Atoi(base); if err != nil { return nil, err }
				end = start
			}
		}
		if start < min || end > max || start > end { return nil, fmt.Errorf("out of range %q", item) }
		for n := start; n <= end; n += step { result[n] = true }
	}
	return result, nil
}
