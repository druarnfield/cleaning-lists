package scheduler

import "time"

// GetWeekStart returns the Monday of the week for a given date
func GetWeekStart(date time.Time) time.Time {
	weekday := int(date.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday should be 7 for this calculation
	}
	daysFromMonday := weekday - 1
	monday := date.AddDate(0, 0, -daysFromMonday)
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
}
