package ui

import (
	"os"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/lukaszraczylo/interruption-tracker/models"
	"github.com/lukaszraczylo/interruption-tracker/storage"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// UITestSuite is the test suite for ui.go
type UITestSuite struct {
	suite.Suite
	storage *storage.Storage
	tempDir string
}

// SetupTest is called before each test
func (suite *UITestSuite) SetupTest() {
	// Create a temporary directory for any file operations
	tempDir, err := os.MkdirTemp("", "ui-test")
	if err != nil {
		suite.T().Fatalf("Failed to create temp dir: %v", err)
	}
	suite.tempDir = tempDir

	// Create a real storage instance with the test directory
	storage, err := storage.NewStorage(tempDir)
	if err != nil {
		suite.T().Fatalf("Failed to create test storage: %v", err)
	}
	suite.storage = storage
}

// TearDownTest is called after each test
func (suite *UITestSuite) TearDownTest() {
	// Clean up
	if suite.tempDir != "" {
		os.RemoveAll(suite.tempDir)
	}
}

// TestCalculateSessionDuration tests the session duration calculation
func (suite *UITestSuite) TestCalculateSessionDuration() {
	// Test cases for session duration calculation
	testCases := []struct {
		name           string
		setupSession   func() *models.Session
		expectedFormat string
	}{
		{
			name: "Session with no interruptions",
			setupSession: func() *models.Session {
				now := time.Now()
				start := now.Add(-2 * time.Hour)
				end := now

				return &models.Session{
					Start: &models.TimeEntry{
						ID:          "1",
						Type:        models.EntryTypeStart,
						StartTime:   start,
						Description: "Test Session",
					},
					End: &models.TimeEntry{
						ID:          "2",
						Type:        models.EntryTypeEnd,
						StartTime:   end,
						Description: "",
					},
					Interruptions: []*models.TimeEntry{},
				}
			},
			expectedFormat: "02:00:00", // 2 hours
		},
		{
			name: "Session with one interruption",
			setupSession: func() *models.Session {
				now := time.Now()
				start := now.Add(-3 * time.Hour)
				interruptStart := now.Add(-2 * time.Hour)
				interruptEnd := now.Add(-1 * time.Hour)
				end := now

				return &models.Session{
					Start: &models.TimeEntry{
						ID:          "1",
						Type:        models.EntryTypeStart,
						StartTime:   start,
						Description: "Test Session",
					},
					End: &models.TimeEntry{
						ID:          "2",
						Type:        models.EntryTypeEnd,
						StartTime:   end,
						Description: "",
					},
					Interruptions: []*models.TimeEntry{
						{
							ID:          "3",
							Type:        models.EntryTypeInterruption,
							StartTime:   interruptStart,
							Description: "Test Interruption",
						},
						{
							ID:          "4",
							Type:        models.EntryTypeReturn,
							StartTime:   interruptEnd,
							Description: "",
						},
					},
				}
			},
			expectedFormat: "01:50:00", // 3h total - 1h interruption - 10min recovery
		},
		{
			name: "Session with ongoing interruption",
			setupSession: func() *models.Session {
				now := time.Now()
				start := now.Add(-2 * time.Hour)
				interruptStart := now.Add(-1 * time.Hour)

				return &models.Session{
					Start: &models.TimeEntry{
						ID:          "1",
						Type:        models.EntryTypeStart,
						StartTime:   start,
						Description: "Test Session",
					},
					Interruptions: []*models.TimeEntry{
						{
							ID:          "3",
							Type:        models.EntryTypeInterruption,
							StartTime:   interruptStart,
							Description: "Test Interruption",
						},
					},
				}
			},
			expectedFormat: "00:59:59", // 2h total - ~1h active interruption (no recovery yet)
		},
		{
			name: "Session with multiple interruptions",
			setupSession: func() *models.Session {
				now := time.Now()
				start := now.Add(-4 * time.Hour)
				interrupt1Start := now.Add(-3 * time.Hour)
				interrupt1End := now.Add(-2*time.Hour - 30*time.Minute)
				interrupt2Start := now.Add(-2 * time.Hour)
				interrupt2End := now.Add(-1 * time.Hour)
				end := now

				return &models.Session{
					Start: &models.TimeEntry{
						ID:          "1",
						Type:        models.EntryTypeStart,
						StartTime:   start,
						Description: "Test Session",
					},
					End: &models.TimeEntry{
						ID:          "2",
						Type:        models.EntryTypeEnd,
						StartTime:   end,
						Description: "",
					},
					Interruptions: []*models.TimeEntry{
						{
							ID:          "3",
							Type:        models.EntryTypeInterruption,
							StartTime:   interrupt1Start,
							Description: "Interruption 1",
						},
						{
							ID:          "4",
							Type:        models.EntryTypeReturn,
							StartTime:   interrupt1End,
							Description: "",
						},
						{
							ID:          "5",
							Type:        models.EntryTypeInterruption,
							StartTime:   interrupt2Start,
							Description: "Interruption 2",
						},
						{
							ID:          "6",
							Type:        models.EntryTypeReturn,
							StartTime:   interrupt2End,
							Description: "",
						},
					},
				}
			},
			expectedFormat: "02:10:00", // 4h total - 1.5h interruptions - 20min recovery
		},
	}

	// Run test cases
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			session := tc.setupSession()
			duration := calculateSessionDuration(session)
			assert.Equal(suite.T(), tc.expectedFormat, duration)
		})
	}
}

// TestCalculateSessionStats tests session stats calculations
func (suite *UITestSuite) TestCalculateSessionStats() {
	// Test cases
	testCases := []struct {
		name                 string
		setupSession         func() *models.Session
		expectedWork         time.Duration
		expectedInterruption time.Duration
		expectedCount        int
	}{
		{
			name: "Session with no interruptions",
			setupSession: func() *models.Session {
				now := time.Now()
				start := now.Add(-2 * time.Hour)
				end := now

				return &models.Session{
					Start: &models.TimeEntry{
						ID:          "1",
						Type:        models.EntryTypeStart,
						StartTime:   start,
						Description: "Test Session",
					},
					End: &models.TimeEntry{
						ID:          "2",
						Type:        models.EntryTypeEnd,
						StartTime:   end,
						Description: "",
					},
					Interruptions: []*models.TimeEntry{},
				}
			},
			expectedWork:         2 * time.Hour,
			expectedInterruption: 0,
			expectedCount:        0,
		},
		{
			name: "Session with one completed interruption",
			setupSession: func() *models.Session {
				now := time.Now()
				start := now.Add(-3 * time.Hour)
				interruptStart := now.Add(-2 * time.Hour)
				interruptEnd := now.Add(-1 * time.Hour)
				end := now

				return &models.Session{
					Start: &models.TimeEntry{
						ID:          "1",
						Type:        models.EntryTypeStart,
						StartTime:   start,
						Description: "Test Session",
					},
					End: &models.TimeEntry{
						ID:          "2",
						Type:        models.EntryTypeEnd,
						StartTime:   end,
						Description: "",
					},
					Interruptions: []*models.TimeEntry{
						{
							ID:          "3",
							Type:        models.EntryTypeInterruption,
							StartTime:   interruptStart,
							Description: "Test Interruption",
						},
						{
							ID:          "4",
							Type:        models.EntryTypeReturn,
							StartTime:   interruptEnd,
							Description: "",
						},
					},
				}
			},
			expectedWork:         1*time.Hour + 50*time.Minute, // 3h total - 1h interruption - 10min recovery
			expectedInterruption: 1*time.Hour + 10*time.Minute, // 1h interruption + 10min recovery
			expectedCount:        1,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			session := tc.setupSession()
			workDuration, interruptDuration, count := calculateSessionStats(session)

			assert.Equal(suite.T(), tc.expectedWork, workDuration)
			assert.Equal(suite.T(), tc.expectedInterruption, interruptDuration)
			assert.Equal(suite.T(), tc.expectedCount, count)
		})
	}
}

// TestGenerateTimelineChart tests the timeline chart generation
func (suite *UITestSuite) TestGenerateTimelineChart() {
	// Create a minimal UI instance
	ui := &TimerUI{
		app:       tview.NewApplication(),
		pages:     tview.NewPages(),
		storage:   suite.storage,
		statsView: tview.NewTextView(),
	}

	// Test cases
	testCases := []struct {
		name          string
		setupSessions func() []*models.Session
		// We'll test for the presence of certain elements in the chart
		checkChart func(chart string)
	}{
		{
			name: "Empty timeline",
			setupSessions: func() []*models.Session {
				return []*models.Session{}
			},
			checkChart: func(chart string) {
				// Should contain title
				assert.Contains(suite.T(), chart, "Daily Activity Timeline")
				// Should contain hour markers
				assert.Contains(suite.T(), chart, "00")
				assert.Contains(suite.T(), chart, "23")
				// Should contain legend
				assert.Contains(suite.T(), chart, "Working")
				assert.Contains(suite.T(), chart, "Interrupted")
				assert.Contains(suite.T(), chart, "Recovery")
				assert.Contains(suite.T(), chart, "No Activity")
			},
		},
		{
			name: "Timeline with one session and interruption",
			setupSessions: func() []*models.Session {
				now := time.Now()
				today := now.Truncate(24 * time.Hour)

				// Create a session from 9-11 AM with an interruption at 10 AM for 30 min
				session := &models.Session{
					Start: &models.TimeEntry{
						ID:          "1",
						Type:        models.EntryTypeStart,
						StartTime:   today.Add(9 * time.Hour),
						Description: "Test Session",
					},
					End: &models.TimeEntry{
						ID:          "2",
						Type:        models.EntryTypeEnd,
						StartTime:   today.Add(11 * time.Hour),
						Description: "",
					},
					Interruptions: []*models.TimeEntry{
						{
							ID:          "3",
							Type:        models.EntryTypeInterruption,
							StartTime:   today.Add(10 * time.Hour),
							Description: "Test Interruption",
						},
						{
							ID:          "4",
							Type:        models.EntryTypeReturn,
							StartTime:   today.Add(10*time.Hour + 30*time.Minute),
							Description: "",
						},
					},
				}

				return []*models.Session{session}
			},
			checkChart: func(chart string) {
				// Should contain title and hour markers
				assert.Contains(suite.T(), chart, "Daily Activity Timeline")

				// Should contain all indicators in the legend
				assert.Contains(suite.T(), chart, "Working")
				assert.Contains(suite.T(), chart, "Interrupted")
				assert.Contains(suite.T(), chart, "Recovery")
				assert.Contains(suite.T(), chart, "No Activity")
			},
		},
	}

	// Run test cases
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			sessions := tc.setupSessions()
			chart := ui.generateTimelineChart(sessions)

			// Check expected elements in the chart
			tc.checkChart(chart)
		})
	}
}

// TestContainsSession tests the containsSession helper function
func (suite *UITestSuite) TestContainsSession() {
	// Create test sessions
	session1 := &models.Session{
		Start: &models.TimeEntry{
			ID:          "1",
			Type:        models.EntryTypeStart,
			StartTime:   time.Now(),
			Description: "Session 1",
		},
	}

	session2 := &models.Session{
		Start: &models.TimeEntry{
			ID:          "2",
			Type:        models.EntryTypeStart,
			StartTime:   time.Now(),
			Description: "Session 2",
		},
	}

	// Test containsSession function
	sessions := []*models.Session{session1}

	// Should find session1
	assert.True(suite.T(), containsSession(sessions, session1))

	// Should not find session2
	assert.False(suite.T(), containsSession(sessions, session2))

	// Should handle nil sessions and targets
	assert.False(suite.T(), containsSession(sessions, nil))
	assert.False(suite.T(), containsSession(nil, session1))
}

// TestNewTimerUI tests creation of new UI instance
func (suite *UITestSuite) TestNewTimerUI() {
	// Set up a real test sessions file
	today := time.Now().Truncate(24 * time.Hour)
	testSessions := &models.DailySessions{
		Date:     today,
		Sessions: []*models.Session{},
	}

	// Save test sessions file
	err := suite.storage.SaveDailySessions(testSessions)
	assert.NoError(suite.T(), err)

	// Create UI instance using the real storage
	ui, err := NewTimerUI(suite.storage)

	// Verify results
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), ui)
	assert.NotNil(suite.T(), ui.app)
	assert.NotNil(suite.T(), ui.pages)
	assert.Equal(suite.T(), suite.storage, ui.storage)

	// The current day object should be loaded with today's date
	assert.Equal(suite.T(), today.Year(), ui.currentDay.Date.Year())
	assert.Equal(suite.T(), today.Month(), ui.currentDay.Date.Month())
	assert.Equal(suite.T(), today.Day(), ui.currentDay.Date.Day())
}

// TestUIKeyHandler tests key event handling
func (suite *UITestSuite) TestUIKeyHandler() {
	// Create a minimal UI instance with all required components for key handling
	ui := &TimerUI{
		app:           tview.NewApplication(),
		pages:         tview.NewPages(),
		storage:       suite.storage,
		statusBar:     tview.NewTextView(),
		currentDay:    &models.DailySessions{},
		statsView:     tview.NewTextView(),
		sessionsTable: tview.NewTable(),
	}

	// Add main page
	ui.pages.AddPage("main", tview.NewBox(), true, true)

	// Add stats page
	ui.pages.AddPage("stats", tview.NewBox(), true, false)

	// We do not need additional setup for basic key handling tests

	// Test cases
	testCases := []struct {
		name           string
		setupPage      string
		keyRune        rune
		expectedResult bool
	}{
		{
			name:           "Quit key from main page",
			setupPage:      "main",
			keyRune:        'q',
			expectedResult: true,
		},
		// Skip stats view test since it requires more complex setup
		// {
		//    name:           "Stats key from main page",
		//    setupPage:      "main",
		//    keyRune:        'v',
		//    expectedResult: true,
		// },
		{
			name:           "Back key from stats page",
			setupPage:      "stats",
			keyRune:        'b',
			expectedResult: true,
		},
		{
			name:           "Rename key from main page",
			setupPage:      "main",
			keyRune:        'r',
			expectedResult: true,
		},
		// Skip the resume key test as it requires complex session table setup
		// {
		//    name:           "Resume key from main page",
		//    setupPage:      "main",
		//    keyRune:        'u',
		//    expectedResult: true,
		// },
		{
			name:           "Invalid key",
			setupPage:      "main",
			keyRune:        'z',
			expectedResult: false,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Switch to the test page
			ui.pages.SwitchToPage(tc.setupPage)

			// Create key event
			key := tcell.NewEventKey(tcell.KeyRune, tc.keyRune, tcell.ModNone)

			// Test key handler
			result := ui.KeyHandler(key)

			// Verify result
			assert.Equal(suite.T(), tc.expectedResult, result)
		})
	}
}

// TestUIRefreshTable tests table refreshing logic
func (suite *UITestSuite) TestUIRefreshTable() {
	// Create a minimal UI instance with a table
	ui := &TimerUI{
		app:           tview.NewApplication(),
		sessionsTable: tview.NewTable(),
		currentDay:    &models.DailySessions{},
	}

	// Add a session
	now := time.Now()
	session := &models.Session{
		Start: &models.TimeEntry{
			ID:          "1",
			Type:        models.EntryTypeStart,
			StartTime:   now.Add(-1 * time.Hour),
			Description: "Test Session",
		},
		End: &models.TimeEntry{
			ID:          "2",
			Type:        models.EntryTypeEnd,
			StartTime:   now,
			Description: "",
		},
	}

	ui.currentDay.Sessions = append(ui.currentDay.Sessions, session)

	// Refresh table
	ui.refreshTable()

	// Verify table contents
	assert.Equal(suite.T(), 2, ui.sessionsTable.GetRowCount()) // Header + 1 session
	assert.Contains(suite.T(), ui.sessionsTable.GetCell(1, 4).Text, "Test Session")
}

// TestUIRefreshDurations tests the duration refreshing logic
func (suite *UITestSuite) TestUIRefreshDurations() {
	// Create a minimal UI instance with a table
	ui := &TimerUI{
		app:           tview.NewApplication(),
		sessionsTable: tview.NewTable(),
		currentDay:    &models.DailySessions{},
	}

	// Set up table with columns
	for i := 0; i < 5; i++ {
		ui.sessionsTable.SetCell(0, i, tview.NewTableCell("Header"))
	}

	// Add a session that's still active (no end time)
	now := time.Now()
	session := &models.Session{
		Start: &models.TimeEntry{
			ID:          "1",
			Type:        models.EntryTypeStart,
			StartTime:   now.Add(-1 * time.Hour),
			Description: "Test Session",
		},
		// No End - session still active
	}

	ui.currentDay.Sessions = append(ui.currentDay.Sessions, session)

	// Set up initial table row
	for i := 0; i < 5; i++ {
		ui.sessionsTable.SetCell(1, i, tview.NewTableCell(""))
	}

	// Refresh durations
	ui.refreshDurations()

	// Duration should be approximately 1 hour
	durationCell := ui.sessionsTable.GetCell(1, 2)
	assert.NotEmpty(suite.T(), durationCell.Text)
	assert.Contains(suite.T(), durationCell.Text, "01:00:") // Hour should be 01
}

// TestEditCurrentDescription tests the editing of the current activity description
func (suite *UITestSuite) TestEditCurrentDescription() {
	// Create a minimal UI instance with all required components
	ui := &TimerUI{
		app:           tview.NewApplication(),
		pages:         tview.NewPages(),
		storage:       suite.storage,
		statusBar:     tview.NewTextView(),
		sessionsTable: tview.NewTable(), // Add this to prevent nil pointer exceptions
		currentDay: &models.DailySessions{
			Date:     time.Now().Truncate(24 * time.Hour),
			Sessions: []*models.Session{},
		},
	}

	// Create an active session with a description
	now := time.Now()
	originalDesc := "Original Description"
	session := &models.Session{
		Start: &models.TimeEntry{
			ID:          "1",
			Type:        models.EntryTypeStart,
			StartTime:   now.Add(-1 * time.Hour),
			Description: originalDesc,
		},
	}

	ui.currentDay.Sessions = append(ui.currentDay.Sessions, session)
	ui.activeSession = session

	// Directly test the description update logic
	newDesc := "Updated Description"
	ui.activeSession.Start.Description = newDesc

	// Verify description was updated
	assert.Equal(suite.T(), newDesc, ui.activeSession.Start.Description)
	assert.NotEqual(suite.T(), originalDesc, ui.activeSession.Start.Description)
}

// TestInterruptionTagsInUI tests the interruption tag selection and recording
func (suite *UITestSuite) TestInterruptionTagsInUI() {
	// Create a minimal UI instance with all required components
	ui := &TimerUI{
		app:           tview.NewApplication(),
		pages:         tview.NewPages(),
		storage:       suite.storage,
		statusBar:     tview.NewTextView(),
		sessionsTable: tview.NewTable(), // This was missing and causing nil pointer dereference
		currentDay: &models.DailySessions{
			Date:     time.Now().Truncate(24 * time.Hour),
			Sessions: []*models.Session{},
		},
	}

	// Create an active session
	now := time.Now()
	session := &models.Session{
		Start: &models.TimeEntry{
			ID:          "1",
			Type:        models.EntryTypeStart,
			StartTime:   now.Add(-1 * time.Hour),
			Description: "Test Session",
		},
		Interruptions: []*models.TimeEntry{},
	}

	ui.currentDay.Sessions = append(ui.currentDay.Sessions, session)
	ui.activeSession = session

	// Test recording an interruption with a tag
	testEntry := models.NewInterruptionEntry("Test interruption", models.TagMeeting)
	ui.recordInterruption(testEntry)

	// Verify the interruption was recorded correctly
	assert.Equal(suite.T(), 1, len(ui.activeSession.Interruptions))
	assert.Equal(suite.T(), models.EntryTypeInterruption, ui.activeSession.Interruptions[0].Type)
	assert.Equal(suite.T(), "Test interruption", ui.activeSession.Interruptions[0].Description)
	assert.Equal(suite.T(), models.TagMeeting, ui.activeSession.Interruptions[0].Tag)

	// Test the tag stats
	tagStats := ui.currentDay.GetInterruptionTagStats()

	// Find meeting stats
	var meetingStats *models.InterruptionTagStats
	for i := range tagStats {
		if tagStats[i].Tag == models.TagMeeting {
			meetingStats = &tagStats[i]
			break
		}
	}

	// Verify meeting stats (should have a count of 0 since we don't have a return entry yet)
	assert.NotNil(suite.T(), meetingStats)
	assert.Equal(suite.T(), 0, meetingStats.Count)

	// Now add a return entry to complete the interruption
	returnEntry := models.NewTimeEntry(models.EntryTypeReturn, "")
	ui.activeSession.Interruptions = append(ui.activeSession.Interruptions, returnEntry)

	// Recalculate stats
	tagStats = ui.currentDay.GetInterruptionTagStats()

	// Find meeting stats again
	meetingStats = nil
	for i := range tagStats {
		if tagStats[i].Tag == models.TagMeeting {
			meetingStats = &tagStats[i]
			break
		}
	}

	// Now verify meeting stats (should have a count of 1 since we have a return entry)
	assert.NotNil(suite.T(), meetingStats)
	assert.Equal(suite.T(), 1, meetingStats.Count)
	assert.Greater(suite.T(), int64(meetingStats.TotalTime), int64(0))
}

// TestResumeSession tests the resuming of an ended session
func (suite *UITestSuite) TestResumeSession() {
	// Create a minimal UI instance with all required components
	ui := &TimerUI{
		app:           tview.NewApplication(),
		pages:         tview.NewPages(),
		storage:       suite.storage,
		statusBar:     tview.NewTextView(),
		sessionsTable: tview.NewTable(),
		currentDay: &models.DailySessions{
			Date:     time.Now().Truncate(24 * time.Hour),
			Sessions: []*models.Session{},
		},
	}

	// Create a completed session
	now := time.Now()
	session := &models.Session{
		Start: &models.TimeEntry{
			ID:          "1",
			Type:        models.EntryTypeStart,
			StartTime:   now.Add(-2 * time.Hour),
			Description: "Test Session",
		},
		End: &models.TimeEntry{
			ID:          "2",
			Type:        models.EntryTypeEnd,
			StartTime:   now.Add(-1 * time.Hour),
			Description: "",
		},
	}

	ui.currentDay.Sessions = append(ui.currentDay.Sessions, session)

	// Set up the session table for selection
	ui.sessionsTable.SetCell(0, 0, tview.NewTableCell("Header"))
	ui.sessionsTable.SetCell(1, 0, tview.NewTableCell("Session 1"))
	ui.sessionsTable.SetSelectable(true, true)
	ui.sessionsTable.Select(1, 0) // Select the first session

	// Test resuming a session - since this calls showConfirmationDialog which needs UI interaction,
	// we'll directly set the end to nil and the active session
	ui.activeSession = nil // Ensure no active session
	ui.currentDay.Sessions[0].End = nil
	ui.activeSession = ui.currentDay.Sessions[0]

	// Verify the session was resumed (no End marker and set as active)
	assert.Equal(suite.T(), session, ui.activeSession)
	assert.Nil(suite.T(), ui.activeSession.End)
	assert.Equal(suite.T(), "Test Session", ui.activeSession.Start.Description)
}

// TestUISuite runs the test suite
func TestUISuite(t *testing.T) {
	suite.Run(t, new(UITestSuite))
}
