package ui

import (
	"fmt"
	"time"

	"github.com/lukaszraczylo/interruption-tracker/models"
)

// calculateSessionDuration calculates the effective duration of a session considering interruptions
// and recovery time. Returns a formatted string in "HH:MM:SS" format.
func calculateSessionDuration(session *models.Session) string {
	if session.Start == nil {
		return ""
	}

	var startTime time.Time = session.Start.StartTime
	var endTime time.Time

	if session.End != nil {
		// Use the recorded end time
		endTime = session.End.StartTime
	} else {
		// Use current time for active sessions
		endTime = time.Now()
	}

	// Calculate total duration (end - start)
	totalDuration := endTime.Sub(startTime)

	// Calculate interruption time
	var interruptionDuration time.Duration
	var recoveryDuration time.Duration
	recoveryTimePerInterruption := 10 * time.Minute

	for i := 0; i < len(session.Interruptions); i += 2 {
		interruptStart := session.Interruptions[i].StartTime

		var interruptEnd time.Time
		if i+1 < len(session.Interruptions) {
			// Use the return time
			interruptEnd = session.Interruptions[i+1].StartTime
			// Add recovery time for completed interruptions
			recoveryDuration += recoveryTimePerInterruption
		} else {
			// For active interruptions, use current time
			interruptEnd = time.Now()
			// No recovery time for active interruptions
		}

		interruptionDuration += interruptEnd.Sub(interruptStart)
	}

	// Adjust recovery time if it would exceed the remaining time
	remainingDuration := totalDuration - interruptionDuration
	if recoveryDuration > remainingDuration {
		recoveryDuration = remainingDuration
	}

	// Effective duration is total time minus interruption time minus recovery time
	effectiveDuration := totalDuration - interruptionDuration - recoveryDuration

	// Format the duration
	hours := int(effectiveDuration.Hours())
	minutes := int(effectiveDuration.Minutes()) % 60
	seconds := int(effectiveDuration.Seconds()) % 60

	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

// computeSessionDuration computes the effective duration of a session
// including time spent in interruptions
func computeSessionDuration(session *models.Session) string {
	if session.Start == nil {
		return ""
	}

	// If we have sub-sessions, use those for accurate duration calculation
	if len(session.SubSessions) > 0 {
		var totalEffectiveDuration time.Duration

		// Process each sub-session
		for _, subSession := range session.SubSessions {
			var subStartTime time.Time = subSession.Start.StartTime
			var subEndTime time.Time

			if subSession.End != nil {
				// Use the recorded end time
				subEndTime = subSession.End.StartTime
			} else {
				// Use current time for active sub-sessions
				subEndTime = time.Now()
			}

			// Calculate total duration for this sub-session
			subTotalDuration := subEndTime.Sub(subStartTime)

			// Calculate interruption time for this sub-session
			var subInterruptionDuration time.Duration
			for i := 0; i < len(subSession.Interruptions); i += 2 {
				interruptStart := subSession.Interruptions[i].StartTime

				var interruptEnd time.Time
				if i+1 < len(subSession.Interruptions) {
					// Use the return time
					interruptEnd = subSession.Interruptions[i+1].StartTime
				} else {
					// For active interruptions, use current time
					interruptEnd = time.Now()
				}

				subInterruptionDuration += interruptEnd.Sub(interruptStart)
			}

			// Effective duration for this sub-session
			subEffectiveDuration := subTotalDuration - subInterruptionDuration
			totalEffectiveDuration += subEffectiveDuration
		}

		// Format the total duration
		hours := int(totalEffectiveDuration.Hours())
		minutes := int(totalEffectiveDuration.Minutes()) % 60
		seconds := int(totalEffectiveDuration.Seconds()) % 60

		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	} else {
		// Legacy behavior for sessions without sub-sessions
		var startTime time.Time = session.Start.StartTime
		var endTime time.Time

		if session.End != nil {
			// Use the recorded end time
			endTime = session.End.StartTime
		} else {
			// Use current time for active sessions
			endTime = time.Now()
		}

		// Calculate total duration (end - start)
		totalDuration := endTime.Sub(startTime)

		// Calculate interruption time
		var interruptionDuration time.Duration
		for i := 0; i < len(session.Interruptions); i += 2 {
			interruptStart := session.Interruptions[i].StartTime

			var interruptEnd time.Time
			if i+1 < len(session.Interruptions) {
				// Use the return time
				interruptEnd = session.Interruptions[i+1].StartTime
			} else {
				// For active interruptions, use current time
				interruptEnd = time.Now()
			}

			interruptionDuration += interruptEnd.Sub(interruptStart)
		}

		// Effective duration is total time minus interruption time
		effectiveDuration := totalDuration - interruptionDuration

		// Format the duration
		hours := int(effectiveDuration.Hours())
		minutes := int(effectiveDuration.Minutes()) % 60
		seconds := int(effectiveDuration.Seconds()) % 60

		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	}
}

// formatDurationHumanReadable formats a duration in a human-readable format
func formatDurationHumanReadable(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}

	seconds := int(d.Seconds()) % 60
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}

	return fmt.Sprintf("%ds", seconds)
}


// createColorGradient returns a color based on a value's position in a range
func createColorGradient(value, min, max float64) string {
	// Normalize to 0-1 range
	normalized := (value - min) / (max - min)

	if normalized < 0 {
		normalized = 0
	} else if normalized > 1 {
		normalized = 1
	}

	// Use tview compatible color names instead of hex codes
	// Map the normalized value to predefined tview colors
	if normalized < 0.2 {
		return "[red]"
	} else if normalized < 0.4 {
		return "[orange]"
	} else if normalized < 0.6 {
		return "[yellow]"
	} else if normalized < 0.8 {
		return "[lime]"
	} else {
		return "[green]"
	}
}

// applyColorToText applies a color to text based on a value's position in a range
func applyColorToText(text string, value, min, max float64) string {
	colorCode := createColorGradient(value, min, max)
	// The color code already includes brackets, so we don't need to add them
	return fmt.Sprintf("%s%s[-]", colorCode, text)
}


