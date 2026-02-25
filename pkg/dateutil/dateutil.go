package dateutil

import (
	"fmt"
	"strings"
	"time"
)

func AsTime(day, month, year int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

func AsTimeWithMonth(day int, month string, year int) time.Time {
	dt, _ := time.Parse("January", month)
	return time.Date(year, dt.Month(), day, 0, 0, 0, 0, time.UTC)
}

func IsDateMatching(t1, t2 time.Time) bool {
	return t1.Year() == t2.Year() && t1.YearDay() == t2.YearDay()
}

func ParseWeekday(s string) (time.Weekday, error) {
	days := map[string]time.Weekday{
		"sunday":    time.Sunday,
		"monday":    time.Monday,
		"tuesday":   time.Tuesday,
		"wednesday": time.Wednesday,
		"thursday":  time.Thursday,
		"friday":    time.Friday,
		"saturday":  time.Saturday,
	}
	day, ok := days[strings.ToLower(s)]
	if !ok {
		return 0, fmt.Errorf("invalid weekday: %q", s)
	}
	return day, nil
}
