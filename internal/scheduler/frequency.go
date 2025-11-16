package scheduler

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseFrequency converts frequency strings into days between occurrences
// Supports formats like: Daily, Weekly, Fortnightly, Monthly, "8 Weekly", "2x/week"
func ParseFrequency(freq string) (float64, error) {
	freq = strings.TrimSpace(strings.ToLower(freq))

	// Handle standard frequencies
	switch freq {
	case "daily":
		return 1, nil
	case "weekly":
		return 7, nil
	case "fortnightly":
		return 14, nil
	case "monthly":
		return 30, nil
	}

	// Handle "N Weekly" pattern (e.g., "8 Weekly", "12 Weekly", "26 Weekly")
	if strings.HasSuffix(freq, "weekly") {
		parts := strings.Fields(freq)
		if len(parts) == 2 {
			n, err := strconv.Atoi(parts[0])
			if err == nil && n > 0 {
				return float64(n * 7), nil
			}
		}
	}

	// Handle "N x/week" or "Nx/week" pattern (e.g., "2x/week", "3 x/week")
	if strings.Contains(freq, "x/week") {
		// Remove spaces around 'x'
		freq = strings.ReplaceAll(freq, " x/week", "x/week")
		freq = strings.ReplaceAll(freq, "x /week", "x/week")
		freq = strings.ReplaceAll(freq, "x/ week", "x/week")

		parts := strings.Split(freq, "x/week")
		if len(parts) == 2 && parts[1] == "" {
			n, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err == nil && n > 0 {
				return 7.0 / float64(n), nil
			}
		}
	}

	// Handle "Every N days" pattern (e.g., "Every 3 days")
	if strings.HasPrefix(freq, "every") {
		parts := strings.Fields(freq)
		if len(parts) == 3 && parts[2] == "days" {
			n, err := strconv.Atoi(parts[1])
			if err == nil && n > 0 {
				return float64(n), nil
			}
		}
	}

	return 0, fmt.Errorf("unknown frequency format: %s", freq)
}

// GetWeeklyMinutes calculates the average weekly minutes for a task
func GetWeeklyMinutes(frequencyDays float64, estimatedMins int) float64 {
	if frequencyDays == 0 {
		return 0
	}
	// Convert to weekly frequency (7 days per week)
	occurrencesPerWeek := 7.0 / frequencyDays
	return occurrencesPerWeek * float64(estimatedMins)
}

// FormatFrequency converts days back to human-readable frequency
func FormatFrequency(days float64) string {
	switch days {
	case 1:
		return "Daily"
	case 7:
		return "Weekly"
	case 14:
		return "Fortnightly"
	case 30:
		return "Monthly"
	default:
		// Check if it's a multiple of 7 (weekly pattern)
		if int(days)%7 == 0 {
			weeks := int(days) / 7
			return fmt.Sprintf("%d Weekly", weeks)
		}
		// Check if it's a fraction of a week (x/week pattern)
		if days < 7 {
			timesPerWeek := int(7.0 / days)
			if float64(timesPerWeek)*days == 7.0 {
				return fmt.Sprintf("%dx/week", timesPerWeek)
			}
		}
		// Generic format
		return fmt.Sprintf("Every %d days", int(days))
	}
}