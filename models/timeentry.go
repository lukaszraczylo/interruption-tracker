package models

import (
	"fmt"
	"time"
)

// EntryType represents the type of time entry
type EntryType string

const (
	// EntryTypeStart represents the start of a work session
	EntryTypeStart EntryType = "START"
	// EntryTypeEnd represents the end of a work session
	EntryTypeEnd EntryType = "END"
	// EntryTypeInterruption represents an interruption during work
	EntryTypeInterruption EntryType = "INTERRUPTION"
	// EntryTypeReturn represents returning from an interruption
	EntryTypeReturn EntryType = "RETURN"
)

// InterruptionTag represents the reason for interruption
type InterruptionTag string

const (
	// TagCall represents a phone call interruption
	TagCall InterruptionTag = "call"
	// TagMeeting represents a meeting interruption
	TagMeeting InterruptionTag = "meeting"
	// TagSpouse represents a spouse/family interruption
	TagSpouse InterruptionTag = "spouse"
	// TagOther represents any other interruption type
	TagOther InterruptionTag = "other"
)

// GetInterruptionTags returns a list of all available interruption tags
func GetInterruptionTags() []InterruptionTag {
	return []InterruptionTag{
		TagCall,
		TagMeeting,
		TagSpouse,
		TagOther,
	}
}

// TimeEntry represents a single time entry in the tracker
type TimeEntry struct {
	ID          string          `json:"id"`
	Type        EntryType       `json:"type"`
	StartTime   time.Time       `json:"start_time"`
	EndTime     time.Time       `json:"end_time,omitempty"`
	Description string          `json:"description,omitempty"`
	Tag         InterruptionTag `json:"tag,omitempty"`
}

// NewTimeEntry creates a new time entry with the given type and description
func NewTimeEntry(entryType EntryType, description string) *TimeEntry {
	now := time.Now()
	return &TimeEntry{
		ID:          fmt.Sprintf("%d", now.UnixNano()),
		Type:        entryType,
		StartTime:   now,
		Description: description,
	}
}

// NewInterruptionEntry creates a new interruption entry with a tag
func NewInterruptionEntry(description string, tag InterruptionTag) *TimeEntry {
	entry := NewTimeEntry(EntryTypeInterruption, description)
	entry.Tag = tag
	return entry
}

// FormatTime formats the time for display
func FormatTime(t time.Time) string {
	return t.Format("15:04:05")
}

// FormatDuration formats the duration between two times
func FormatDuration(start, end time.Time) string {
	duration := end.Sub(start)
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

// SubSession represents a continuous period of work within a session
type SubSession struct {
	Start         *TimeEntry   `json:"start"`
	End           *TimeEntry   `json:"end,omitempty"`
	Interruptions []*TimeEntry `json:"interruptions,omitempty"`
}

// Session represents a complete work session that may contain multiple sub-sessions
type Session struct {
	ID            string        `json:"id"`                      // Unique ID for this session
	Start         *TimeEntry    `json:"start"`                   // First start time of the task
	End           *TimeEntry    `json:"end,omitempty"`           // Most recent end time, omitted if active
	SubSessions   []*SubSession `json:"sub_sessions"`            // List of continuous work periods
	Interruptions []*TimeEntry  `json:"interruptions,omitempty"` // For backward compatibility
}

// DailySessions represents all sessions for a single day
type DailySessions struct {
	Date     time.Time  `json:"date"`
	Sessions []*Session `json:"sessions"`
}

// NewDailySessions creates a new DailySessions for the current day
func NewDailySessions() *DailySessions {
	return &DailySessions{
		Date:     time.Now().Truncate(24 * time.Hour),
		Sessions: []*Session{},
	}
}

// NewSession creates a new session with the given start entry and initializes an empty sub-sessions array
func NewSession(startEntry *TimeEntry) *Session {
	// Generate a unique ID for the session
	now := time.Now()
	sessionID := fmt.Sprintf("sess_%d", now.UnixNano())

	session := &Session{
		ID:          sessionID,
		Start:       startEntry,
		SubSessions: []*SubSession{},
	}

	// Create the first sub-session
	firstSubSession := &SubSession{
		Start:         startEntry,
		Interruptions: []*TimeEntry{},
	}
	session.SubSessions = append(session.SubSessions, firstSubSession)

	return session
}

// GetStats calculates statistics for the daily sessions
func (ds *DailySessions) GetStats() (totalWorkDuration, totalInterruptionDuration time.Duration, interruptionCount int) {
	for _, session := range ds.Sessions {
		// If the session has sub-sessions, use those for accurate duration calculation
		if len(session.SubSessions) > 0 {
			for _, subSession := range session.SubSessions {
				if subSession.Start != nil {
					var endTime time.Time

					if subSession.End != nil {
						endTime = subSession.End.StartTime
					} else {
						// For active sub-sessions, use current time
						endTime = time.Now()
					}

					subSessionDuration := endTime.Sub(subSession.Start.StartTime)
					interruptionDuration := time.Duration(0)

					// Calculate interruption time within this sub-session
					for i := 0; i < len(subSession.Interruptions); i += 2 {
						if i+1 < len(subSession.Interruptions) {
							interruptionStart := subSession.Interruptions[i].StartTime
							interruptionEnd := subSession.Interruptions[i+1].StartTime
							interruptionDuration += interruptionEnd.Sub(interruptionStart)
						}
					}

					totalWorkDuration += subSessionDuration - interruptionDuration
					totalInterruptionDuration += interruptionDuration
					interruptionCount += len(subSession.Interruptions) / 2
				}
			}
		} else {
			// Backward compatibility for sessions without sub-sessions
			if session.Start != nil && session.End != nil {
				sessionDuration := session.End.StartTime.Sub(session.Start.StartTime)
				interruptionDuration := time.Duration(0)

				for i := 0; i < len(session.Interruptions); i += 2 {
					if i+1 < len(session.Interruptions) {
						interruptionStart := session.Interruptions[i].StartTime
						interruptionEnd := session.Interruptions[i+1].StartTime
						interruptionDuration += interruptionEnd.Sub(interruptionStart)
					}
				}

				totalWorkDuration += sessionDuration - interruptionDuration
				totalInterruptionDuration += interruptionDuration
				interruptionCount += len(session.Interruptions) / 2
			}
		}
	}

	return totalWorkDuration, totalInterruptionDuration, interruptionCount
}

// InterruptionTagStats represents statistics for a specific interruption tag
type InterruptionTagStats struct {
	Tag               InterruptionTag
	Count             int
	TotalTime         time.Duration // Pure interruption time without recovery
	RecoveryTime      time.Duration // Separate recovery time
	TotalWithRecovery time.Duration // Combined total of interruption + recovery
	AverageTime       time.Duration // Average pure interruption time
}

// GetInterruptionTagStats calculates statistics for different types of interruptions
func (ds *DailySessions) GetInterruptionTagStats() []InterruptionTagStats {
	// Create a map to collect stats for each tag
	statsMap := make(map[InterruptionTag]*InterruptionTagStats)

	// Initialize stats for all possible tags
	for _, tag := range GetInterruptionTags() {
		statsMap[tag] = &InterruptionTagStats{Tag: tag}
	}

	// Collect data from all sessions
	for _, session := range ds.Sessions {
		for i := 0; i < len(session.Interruptions); i += 2 {
			// Only count completed interruptions
			if i+1 < len(session.Interruptions) {
				interruption := session.Interruptions[i]
				returnEntry := session.Interruptions[i+1]

				// Use the tag or fallback to "other" if not set
				tag := interruption.Tag
				if tag == "" {
					tag = TagOther
				}

				// Get or create stats for this tag
				stats := statsMap[tag]

				// Update the stats
				stats.Count++

				// Calculate interruption duration
				interruptDuration := returnEntry.StartTime.Sub(interruption.StartTime)

				// Keep track of pure interruption time
				stats.TotalTime += interruptDuration

				// Standard recovery period
				recoveryTime := 10 * time.Minute
				stats.RecoveryTime += recoveryTime

				// Combined total with recovery
				stats.TotalWithRecovery += interruptDuration + recoveryTime
			}
		}
	}

	// Convert map to slice and calculate averages
	result := make([]InterruptionTagStats, 0, len(statsMap))
	for _, stats := range statsMap {
		if stats.Count > 0 {
			stats.AverageTime = stats.TotalTime / time.Duration(stats.Count)
		}
		result = append(result, *stats)
	}

	return result
}
