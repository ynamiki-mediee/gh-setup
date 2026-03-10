package milestone

import (
	"fmt"
	"math"
	"time"
)

// NextMonday returns the next Monday from today as "YYYY-MM-DD".
// If today is Monday, it returns the following Monday (always in the future).
func NextMonday() string {
	today := time.Now()
	daysUntilMonday := (int(time.Monday) - int(today.Weekday()) + 7) % 7
	if daysUntilMonday == 0 {
		daysUntilMonday = 7
	}
	nextMon := today.AddDate(0, 0, daysUntilMonday)
	return nextMon.Format("2006-01-02")
}

// WeeksUntilEndOfYear calculates the number of weeks from startDate to Dec 31
// of that year. Returns max(1, ceil(diffDays / 7)).
func WeeksUntilEndOfYear(startDate string) int {
	t, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return 1
	}
	endOfYear := time.Date(t.Year(), 12, 31, 0, 0, 0, 0, time.UTC)
	diffDays := endOfYear.Sub(t).Hours() / 24
	weeks := int(math.Ceil(diffDays / 7))
	if weeks < 1 {
		return 1
	}
	return weeks
}

// ISOWeek returns the ISO 8601 week number for the given date.
func ISOWeek(date time.Time) int {
	_, week := date.ISOWeek()
	return week
}

// ToUtcDueOn converts a "YYYY-MM-DD" date string and timezone name to a UTC
// ISO 8601 datetime string. The time is set to 23:59:59 in the given timezone,
// then converted to UTC.
func ToUtcDueOn(dateStr, timezone string) (string, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return "", fmt.Errorf("invalid timezone %q: %w", timezone, err)
	}

	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return "", fmt.Errorf("invalid date %q: %w", dateStr, err)
	}

	localTime := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, loc)
	utcTime := localTime.UTC()

	return utcTime.Format("2006-01-02T15:04:05Z"), nil
}
