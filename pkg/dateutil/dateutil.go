package dateutil

import "time"

func AsTime(day, month, year int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

func AsTimeWithMonth(day int, month string, year int) time.Time {
	dt, _ := time.Parse("January", month)
	return time.Date(year, dt.Month(), day, 0, 0, 0, 0, time.UTC)
}
