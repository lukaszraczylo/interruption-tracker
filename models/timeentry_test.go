package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// TimeEntryTestSuite is the test suite for timeentry.go
type TimeEntryTestSuite struct {
	suite.Suite
}

// SetupTest is called before each test
func (suite *TimeEntryTestSuite) SetupTest() {
	// No setup required for now
}

// TestNewTimeEntry verifies that a new time entry is created correctly
func (suite *TimeEntryTestSuite) TestNewTimeEntry() {
	description := "Test Entry"
	entryType := EntryTypeStart

	entry := NewTimeEntry(entryType, description)

	assert.NotNil(suite.T(), entry)
	assert.Equal(suite.T(), entryType, entry.Type)
	assert.Equal(suite.T(), description, entry.Description)
	assert.NotEmpty(suite.T(), entry.ID)

	// Verify that the time is reasonably close to now
	now := time.Now()
	timeDiff := now.Sub(entry.StartTime)
	assert.Less(suite.T(), timeDiff.Milliseconds(), int64(1000)) // Within 1 second
}

// TestFormatTime tests the time formatting function
func (suite *TimeEntryTestSuite) TestFormatTime() {
	// Test with a fixed time
	testTime := time.Date(2025, 3, 8, 14, 30, 45, 0, time.Local)
	formatted := FormatTime(testTime)

	// Should be in format HH:MM:SS
	assert.Equal(suite.T(), "14:30:45", formatted)
}

// TestFormatDuration tests the duration formatting function
func (suite *TimeEntryTestSuite) TestFormatDuration() {
	// Test with a fixed start and end time
	start := time.Date(2025, 3, 8, 14, 30, 0, 0, time.Local)
	end := time.Date(2025, 3, 8, 16, 45, 30, 0, time.Local)

	formatted := FormatDuration(start, end)

	// Duration should be 2 hours, 15 minutes, 30 seconds
	assert.Equal(suite.T(), "02:15:30", formatted)
}

// TestDailySessionsNewAndGetStats tests the creation of daily sessions and statistics calculation
func (suite *TimeEntryTestSuite) TestDailySessionsNewAndGetStats() {
	// Create a new daily sessions object
	dailySessions := NewDailySessions()

	assert.NotNil(suite.T(), dailySessions)
	assert.Empty(suite.T(), dailySessions.Sessions)

	// Ensure date is set to today with time truncated
	now := time.Now().Truncate(24 * time.Hour)
	assert.Equal(suite.T(), now.Year(), dailySessions.Date.Year())
	assert.Equal(suite.T(), now.Month(), dailySessions.Date.Month())
	assert.Equal(suite.T(), now.Day(), dailySessions.Date.Day())
	assert.Equal(suite.T(), 0, dailySessions.Date.Hour())
	assert.Equal(suite.T(), 0, dailySessions.Date.Minute())
	assert.Equal(suite.T(), 0, dailySessions.Date.Second())

	// Calculate stats for empty sessions list
	workDuration, interruptionDuration, interruptionCount := dailySessions.GetStats()
	assert.Equal(suite.T(), time.Duration(0), workDuration)
	assert.Equal(suite.T(), time.Duration(0), interruptionDuration)
	assert.Equal(suite.T(), 0, interruptionCount)
}

// TestSessionStats tests session statistics calculations
func (suite *TimeEntryTestSuite) TestSessionStats() {
	// Test table with different session scenarios
	tests := []struct {
		name              string
		setupSession      func() *Session
		expectedWork      time.Duration
		expectedInterrupt time.Duration
		expectedCount     int
	}{
		{
			name: "No interruptions",
			setupSession: func() *Session {
				now := time.Now()
				start := &TimeEntry{
					ID:          "1",
					Type:        EntryTypeStart,
					StartTime:   now.Add(-1 * time.Hour),
					Description: "Start",
				}
				end := &TimeEntry{
					ID:          "2",
					Type:        EntryTypeEnd,
					StartTime:   now,
					Description: "",
				}
				return &Session{
					Start:         start,
					End:           end,
					Interruptions: []*TimeEntry{},
				}
			},
			expectedWork:      1 * time.Hour,
			expectedInterrupt: 0,
			expectedCount:     0,
		},
		{
			name: "One interruption",
			setupSession: func() *Session {
				now := time.Now()
				start := &TimeEntry{
					ID:          "1",
					Type:        EntryTypeStart,
					StartTime:   now.Add(-2 * time.Hour),
					Description: "Start",
				}
				end := &TimeEntry{
					ID:          "2",
					Type:        EntryTypeEnd,
					StartTime:   now,
					Description: "",
				}
				interrupt := &TimeEntry{
					ID:          "3",
					Type:        EntryTypeInterruption,
					StartTime:   now.Add(-1 * time.Hour),
					Description: "Interruption",
				}
				resume := &TimeEntry{
					ID:          "4",
					Type:        EntryTypeReturn,
					StartTime:   now.Add(-30 * time.Minute),
					Description: "",
				}
				return &Session{
					Start:         start,
					End:           end,
					Interruptions: []*TimeEntry{interrupt, resume},
				}
			},
			expectedWork:      1*time.Hour + 30*time.Minute, // 2h total - 30m interruption
			expectedInterrupt: 30 * time.Minute,
			expectedCount:     1,
		},
		{
			name: "Multiple interruptions",
			setupSession: func() *Session {
				now := time.Now()
				start := &TimeEntry{
					ID:          "1",
					Type:        EntryTypeStart,
					StartTime:   now.Add(-3 * time.Hour),
					Description: "Start",
				}
				end := &TimeEntry{
					ID:          "2",
					Type:        EntryTypeEnd,
					StartTime:   now,
					Description: "",
				}
				interrupt1 := &TimeEntry{
					ID:          "3",
					Type:        EntryTypeInterruption,
					StartTime:   now.Add(-2 * time.Hour),
					Description: "Interruption 1",
				}
				resume1 := &TimeEntry{
					ID:          "4",
					Type:        EntryTypeReturn,
					StartTime:   now.Add(-1*time.Hour - 30*time.Minute),
					Description: "",
				}
				interrupt2 := &TimeEntry{
					ID:          "5",
					Type:        EntryTypeInterruption,
					StartTime:   now.Add(-1 * time.Hour),
					Description: "Interruption 2",
				}
				resume2 := &TimeEntry{
					ID:          "6",
					Type:        EntryTypeReturn,
					StartTime:   now.Add(-30 * time.Minute),
					Description: "",
				}
				return &Session{
					Start:         start,
					End:           end,
					Interruptions: []*TimeEntry{interrupt1, resume1, interrupt2, resume2},
				}
			},
			expectedWork:      2 * time.Hour,
			expectedInterrupt: 1 * time.Hour,
			expectedCount:     2,
		},
	}

	// Create a new daily sessions object
	dailySessions := NewDailySessions()

	// Run each test case
	for _, tc := range tests {
		suite.Run(tc.name, func() {
			session := tc.setupSession()
			dailySessions.Sessions = []*Session{session}

			workDuration, interruptionDuration, interruptionCount := dailySessions.GetStats()

			assert.Equal(suite.T(), tc.expectedWork, workDuration)
			assert.Equal(suite.T(), tc.expectedInterrupt, interruptionDuration)
			assert.Equal(suite.T(), tc.expectedCount, interruptionCount)
		})
	}
}

// TestInterruptionTagFunctions tests the InterruptionTag functionality
func (suite *TimeEntryTestSuite) TestInterruptionTagFunctions() {
	// Test GetInterruptionTags returns all expected tags
	tags := GetInterruptionTags()
	assert.Equal(suite.T(), 4, len(tags))
	assert.Contains(suite.T(), tags, TagCall)
	assert.Contains(suite.T(), tags, TagMeeting)
	assert.Contains(suite.T(), tags, TagSpouse)
	assert.Contains(suite.T(), tags, TagOther)

	// Test tag string values
	assert.Equal(suite.T(), "call", string(TagCall))
	assert.Equal(suite.T(), "meeting", string(TagMeeting))
	assert.Equal(suite.T(), "spouse", string(TagSpouse))
	assert.Equal(suite.T(), "other", string(TagOther))
}

// TestNewInterruptionEntry tests creation of interruption entries with tags
func (suite *TimeEntryTestSuite) TestNewInterruptionEntry() {
	description := "Test Interruption"
	tag := TagMeeting

	entry := NewInterruptionEntry(description, tag)

	assert.NotNil(suite.T(), entry)
	assert.Equal(suite.T(), EntryTypeInterruption, entry.Type)
	assert.Equal(suite.T(), description, entry.Description)
	assert.Equal(suite.T(), tag, entry.Tag)
	assert.NotEmpty(suite.T(), entry.ID)

	// Verify that the time is reasonably close to now
	now := time.Now()
	timeDiff := now.Sub(entry.StartTime)
	assert.Less(suite.T(), timeDiff.Milliseconds(), int64(1000)) // Within 1 second
}

// TestGetInterruptionTagStats tests the interruption tag statistics function
func (suite *TimeEntryTestSuite) TestGetInterruptionTagStats() {
	// Create a test daily sessions object with tagged interruptions
	now := time.Now()
	dailySessions := NewDailySessions()

	// Create a session with multiple tagged interruptions
	session := &Session{
		Start: &TimeEntry{
			ID:          "1",
			Type:        EntryTypeStart,
			StartTime:   now.Add(-3 * time.Hour),
			Description: "Start",
		},
		End: &TimeEntry{
			ID:          "2",
			Type:        EntryTypeEnd,
			StartTime:   now,
			Description: "",
		},
		Interruptions: []*TimeEntry{
			// Call interruption (30 minutes)
			{
				ID:          "3",
				Type:        EntryTypeInterruption,
				StartTime:   now.Add(-2 * time.Hour),
				Description: "Call interruption",
				Tag:         TagCall,
			},
			{
				ID:          "4",
				Type:        EntryTypeReturn,
				StartTime:   now.Add(-2*time.Hour + 30*time.Minute),
				Description: "",
			},
			// Meeting interruption (45 minutes)
			{
				ID:          "5",
				Type:        EntryTypeInterruption,
				StartTime:   now.Add(-1 * time.Hour),
				Description: "Meeting interruption",
				Tag:         TagMeeting,
			},
			{
				ID:          "6",
				Type:        EntryTypeReturn,
				StartTime:   now.Add(-1*time.Hour + 45*time.Minute),
				Description: "",
			},
		},
	}

	dailySessions.Sessions = []*Session{session}

	// Get the tag stats
	tagStats := dailySessions.GetInterruptionTagStats()

	// Should have stats for all tag types, but only 2 with count > 0
	assert.Equal(suite.T(), 4, len(tagStats))

	// Find the call and meeting stats
	var callStats, meetingStats *InterruptionTagStats
	for i := range tagStats {
		if tagStats[i].Tag == TagCall {
			callStats = &tagStats[i]
		} else if tagStats[i].Tag == TagMeeting {
			meetingStats = &tagStats[i]
		}
	}

	// Verify call stats
	// Verify call stats (30 min interruption + 10 min recovery)
	assert.NotNil(suite.T(), callStats)
	assert.Equal(suite.T(), 1, callStats.Count)
	assert.Equal(suite.T(), 30*time.Minute, callStats.TotalTime)
	assert.Equal(suite.T(), 10*time.Minute, callStats.RecoveryTime)
	assert.Equal(suite.T(), 40*time.Minute, callStats.TotalWithRecovery)
	assert.Equal(suite.T(), 30*time.Minute, callStats.AverageTime)

	// Verify meeting stats (45 min interruption + 10 min recovery)
	assert.NotNil(suite.T(), meetingStats)
	assert.Equal(suite.T(), 1, meetingStats.Count)
	assert.Equal(suite.T(), 45*time.Minute, meetingStats.TotalTime)
	assert.Equal(suite.T(), 10*time.Minute, meetingStats.RecoveryTime)
	assert.Equal(suite.T(), 55*time.Minute, meetingStats.TotalWithRecovery)
	assert.Equal(suite.T(), 45*time.Minute, meetingStats.AverageTime)
}

// TestTimeEntrySuite runs the test suite
func TestTimeEntrySuite(t *testing.T) {
	suite.Run(t, new(TimeEntryTestSuite))
}
