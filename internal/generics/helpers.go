package generics

import (
	"fmt"
	"time"
)

// FormatTimeAgo renders a duration as a short relative-time string (e.g. "5m ago").
func FormatTimeAgo(d time.Duration) string {
	return formatTimeAgo(d)
}

// formatTimeAgo renders a duration as a short relative-time string (e.g. "5m ago").
// Negative durations (clock skew, future timestamps) are treated as "just now".
func formatTimeAgo(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d/time.Minute))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d/time.Hour))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d/(24*time.Hour)))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dw ago", int(d/(7*24*time.Hour)))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo ago", int(d/(30*24*time.Hour)))
	default:
		return fmt.Sprintf("%dy ago", int(d/(365*24*time.Hour)))
	}
}

