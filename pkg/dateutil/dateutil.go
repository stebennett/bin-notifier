package dateutil

import (
	"fmt"
	"strings"
	"sync"
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

func normalizeToUTCMidnight(t time.Time) time.Time {
	y, m, d := t.In(time.UTC).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func IsOnWeek(referenceDate, targetDate time.Time, everyNWeeks int) bool {
	if everyNWeeks <= 0 {
		return false
	}
	ref := normalizeToUTCMidnight(referenceDate)
	target := normalizeToUTCMidnight(targetDate)
	days := int(target.Sub(ref).Hours() / 24)
	if days < 0 {
		days = -days
	}
	weeks := days / 7
	return weeks%everyNWeeks == 0
}

// london is loaded lazily so the lookup happens at first use, after the
// `_ "time/tzdata"` blank import (added to the binary entry points) has registered
// the embedded zone database. UTC is a defensive fallback that is unreachable once
// that embedded database is present; note Europe/London observes BST (+1) in summer,
// so UTC is not equivalent year-round.
var london = sync.OnceValue(func() *time.Location {
	loc, err := time.LoadLocation("Europe/London")
	if err != nil {
		return time.UTC
	}
	return loc
})

// London returns the Europe/London location used for all calendar-date logic.
func London() *time.Location { return london() }

// TodayString returns the current calendar date in Europe/London as YYYY-MM-DD.
func TodayString() string { return time.Now().In(London()).Format("2006-01-02") }

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
