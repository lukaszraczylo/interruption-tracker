package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lukaszraczylo/interruption-tracker/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// StorageTestSuite is the test suite for storage.go
type StorageTestSuite struct {
	suite.Suite
	testDir string
	storage *Storage
}

// SetupTest is called before each test
func (suite *StorageTestSuite) SetupTest() {
	// Create a temporary directory for test
	tempDir, err := os.MkdirTemp("", "interruption-tracker-test")
	assert.NoError(suite.T(), err)
	suite.testDir = tempDir

	// Create a storage instance with the test directory
	storage, err := NewStorage(tempDir)
	assert.NoError(suite.T(), err)
	suite.storage = storage
}

// TearDownTest is called after each test
func (suite *StorageTestSuite) TearDownTest() {
	// Clean up test directory
	if suite.testDir != "" {
		os.RemoveAll(suite.testDir)
	}
}

// TestNewStorage tests creating a new storage instance
func (suite *StorageTestSuite) TestNewStorage() {
	// Test creating with a specific directory
	storage1, err := NewStorage(suite.testDir)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), storage1)
	assert.Equal(suite.T(), suite.testDir, storage1.dataDir)

	// Test creating with empty string (should use home directory)
	storage2, err := NewStorage("")
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), storage2)

	homeDir, err := os.UserHomeDir()
	assert.NoError(suite.T(), err)
	expectedPath := filepath.Join(homeDir, ".interruption-tracker")
	assert.Equal(suite.T(), expectedPath, storage2.dataDir)
}

// TestGetFilePath tests file path generation
func (suite *StorageTestSuite) TestGetFilePath() {
	testDate := time.Date(2025, 3, 8, 0, 0, 0, 0, time.Local)
	expectedPath := filepath.Join(suite.testDir, "sessions_2025-03-08.json")

	actualPath := suite.storage.getFilePath(testDate)
	assert.Equal(suite.T(), expectedPath, actualPath)
}

// TestSaveAndLoadDailySessions tests saving and loading daily sessions
func (suite *StorageTestSuite) TestSaveAndLoadDailySessions() {
	// Create test data
	testDate := time.Date(2025, 3, 8, 0, 0, 0, 0, time.Local)
	dailySession := &models.DailySessions{
		Date: testDate,
		Sessions: []*models.Session{
			{
				Start: &models.TimeEntry{
					ID:          "1",
					Type:        models.EntryTypeStart,
					StartTime:   testDate.Add(8 * time.Hour),
					Description: "Test Session",
				},
				End: &models.TimeEntry{
					ID:          "2",
					Type:        models.EntryTypeEnd,
					StartTime:   testDate.Add(10 * time.Hour),
					Description: "",
				},
				Interruptions: []*models.TimeEntry{
					{
						ID:          "3",
						Type:        models.EntryTypeInterruption,
						StartTime:   testDate.Add(9 * time.Hour),
						Description: "Test Interruption",
					},
					{
						ID:          "4",
						Type:        models.EntryTypeReturn,
						StartTime:   testDate.Add(9*time.Hour + 30*time.Minute),
						Description: "",
					},
				},
			},
		},
	}

	// Test saving
	err := suite.storage.SaveDailySessions(dailySession)
	assert.NoError(suite.T(), err)

	// Verify file exists
	filePath := suite.storage.getFilePath(testDate)
	_, err = os.Stat(filePath)
	assert.NoError(suite.T(), err)

	// Test loading
	loadedSessions, err := suite.storage.LoadDailySessions(testDate)
	assert.NoError(suite.T(), err)

	// Verify loaded data matches original (comparing only date, month, year)
	assert.Equal(suite.T(), testDate.Day(), loadedSessions.Date.Day())
	assert.Equal(suite.T(), testDate.Month(), loadedSessions.Date.Month())
	assert.Equal(suite.T(), testDate.Year(), loadedSessions.Date.Year())
	assert.Len(suite.T(), loadedSessions.Sessions, 1)

	// Test descriptions and timestamps
	assert.Equal(suite.T(), "Test Session", loadedSessions.Sessions[0].Start.Description)
	assert.Equal(suite.T(), "Test Interruption", loadedSessions.Sessions[0].Interruptions[0].Description)
	assert.Equal(suite.T(), testDate.Add(8*time.Hour).Unix(), loadedSessions.Sessions[0].Start.StartTime.Unix())
	assert.Equal(suite.T(), testDate.Add(10*time.Hour).Unix(), loadedSessions.Sessions[0].End.StartTime.Unix())
}

// TestLoadNonExistentDailySessions tests loading sessions for a day that has no data
func (suite *StorageTestSuite) TestLoadNonExistentDailySessions() {
	// Use a date that doesn't have any data
	testDate := time.Date(2020, 1, 1, 0, 0, 0, 0, time.Local)

	// Load sessions - should return empty sessions, not an error
	sessions, err := suite.storage.LoadDailySessions(testDate)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), sessions)
	assert.Empty(suite.T(), sessions.Sessions)
	assert.Equal(suite.T(), testDate.Truncate(24*time.Hour), sessions.Date)
}

// TestGetDateRange tests date range calculations for different range types
func (suite *StorageTestSuite) TestGetDateRange() {
	// Store current time for consistent testing
	now := time.Now()
	today := now.Truncate(24 * time.Hour)

	// Test cases
	testCases := []struct {
		name          string
		rangeType     string
		expectedStart time.Time
		expectedEnd   time.Time
		expectError   bool
	}{
		{
			name:          "Day range",
			rangeType:     "day",
			expectedStart: today,
			expectedEnd:   today,
			expectError:   false,
		},
		{
			name:      "Week range",
			rangeType: "week",
			// Start date will be calculated based on current weekday (Monday of current week)
			expectedEnd: today,
			expectError: false,
		},
		{
			name:      "Month range",
			rangeType: "month",
			// Start date will be 1st of current month
			expectedStart: time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location()),
			expectedEnd:   today,
			expectError:   false,
		},
		{
			name:        "Invalid range",
			rangeType:   "invalid",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			start, end, err := suite.storage.GetDateRange(tc.rangeType)

			if tc.expectError {
				assert.Error(suite.T(), err)
			} else {
				assert.NoError(suite.T(), err)

				if tc.rangeType == "week" {
					// For week test, calculate expected start dynamically
					weekday := int(now.Weekday())
					if weekday == 0 { // Sunday
						weekday = 7
					}
					expectedStart := today.AddDate(0, 0, -(weekday - 1))
					assert.Equal(suite.T(), expectedStart, start)
				} else {
					assert.Equal(suite.T(), tc.expectedStart, start)
				}

				assert.Equal(suite.T(), tc.expectedEnd, end)
			}
		})
	}
}

// TestGetStats tests statistics calculation across date ranges
func (suite *StorageTestSuite) TestGetStats() {
	// Create test data for multiple days
	today := time.Now().Truncate(24 * time.Hour)
	yesterday := today.AddDate(0, 0, -1)

	// Create sessions for today
	todaySessions := &models.DailySessions{
		Date: today,
		Sessions: []*models.Session{
			{
				Start: &models.TimeEntry{
					ID:          "1",
					Type:        models.EntryTypeStart,
					StartTime:   today.Add(8 * time.Hour),
					Description: "Today Session",
				},
				End: &models.TimeEntry{
					ID:          "2",
					Type:        models.EntryTypeEnd,
					StartTime:   today.Add(10 * time.Hour),
					Description: "",
				},
				// No interruptions
			},
		},
	}

	// Create sessions for yesterday
	yesterdaySessions := &models.DailySessions{
		Date: yesterday,
		Sessions: []*models.Session{
			{
				Start: &models.TimeEntry{
					ID:          "3",
					Type:        models.EntryTypeStart,
					StartTime:   yesterday.Add(9 * time.Hour),
					Description: "Yesterday Session",
				},
				End: &models.TimeEntry{
					ID:          "4",
					Type:        models.EntryTypeEnd,
					StartTime:   yesterday.Add(12 * time.Hour),
					Description: "",
				},
				Interruptions: []*models.TimeEntry{
					{
						ID:          "5",
						Type:        models.EntryTypeInterruption,
						StartTime:   yesterday.Add(10 * time.Hour),
						Description: "Interruption",
					},
					{
						ID:          "6",
						Type:        models.EntryTypeReturn,
						StartTime:   yesterday.Add(11 * time.Hour),
						Description: "",
					},
				},
			},
		},
	}

	// Save sessions to storage
	err := suite.storage.SaveDailySessions(todaySessions)
	assert.NoError(suite.T(), err)

	err = suite.storage.SaveDailySessions(yesterdaySessions)
	assert.NoError(suite.T(), err)

	// Test getting stats for day
	workDay, interruptDay, countDay, err := suite.storage.GetStats("day")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 2*time.Hour, workDay)           // 2 hours of work today
	assert.Equal(suite.T(), time.Duration(0), interruptDay) // No interruptions today
	assert.Equal(suite.T(), 0, countDay)

	// Test getting stats for week
	workWeek, interruptWeek, countWeek, err := suite.storage.GetStats("week")
	assert.NoError(suite.T(), err)
	// Should include both today and yesterday
	assert.Equal(suite.T(), 4*time.Hour, workWeek)      // 2h from today + 2h from yesterday
	assert.Equal(suite.T(), 1*time.Hour, interruptWeek) // 1h interruption from yesterday
	assert.Equal(suite.T(), 1, countWeek)               // 1 interruption from yesterday
}

// TestListAvailableDays tests listing days with tracking data
func (suite *StorageTestSuite) TestListAvailableDays() {
	// Create test data for multiple days
	day1 := time.Date(2025, 3, 1, 0, 0, 0, 0, time.Local)
	day2 := time.Date(2025, 3, 2, 0, 0, 0, 0, time.Local)

	// Create empty session objects
	sessions1 := &models.DailySessions{Date: day1, Sessions: []*models.Session{}}
	sessions2 := &models.DailySessions{Date: day2, Sessions: []*models.Session{}}

	// Save sessions to storage
	err := suite.storage.SaveDailySessions(sessions1)
	assert.NoError(suite.T(), err)

	err = suite.storage.SaveDailySessions(sessions2)
	assert.NoError(suite.T(), err)

	// List available days
	days, err := suite.storage.ListAvailableDays()
	assert.NoError(suite.T(), err)

	// Should have two days
	assert.Len(suite.T(), days, 2)

	// Create a map for easy lookup of dates
	dateMap := make(map[string]bool)
	for _, d := range days {
		dateMap[d.Format("2006-01-02")] = true
	}

	// Both test days should be in the list
	assert.True(suite.T(), dateMap["2025-03-01"])
	assert.True(suite.T(), dateMap["2025-03-02"])
}

// TestStorageSuite runs the test suite
func TestStorageSuite(t *testing.T) {
	suite.Run(t, new(StorageTestSuite))
}
