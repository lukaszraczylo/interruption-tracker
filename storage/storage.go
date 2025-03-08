package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/lukaszraczylo/interruption-tracker/config"
	"github.com/lukaszraczylo/interruption-tracker/models"
)

// Storage handles persistence of time entries
type Storage struct {
	dataDir           string
	backupEnabled     bool
	backupInterval    int // Days between backups
	encryptionEnabled bool
	encryptionKey     []byte
	config            *config.Config
}

// NewStorage creates a new storage instance
func NewStorage(customDataDir string) (*Storage, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	dataDir := cfg.DataDirectory
	if customDataDir != "" {
		dataDir = customDataDir
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Generate encryption key if needed
	var encryptionKey []byte
	if cfg.EnableEncryption {
		if cfg.EncryptionKey != "" {
			// Use provided key
			hash := sha256.Sum256([]byte(cfg.EncryptionKey))
			encryptionKey = hash[:]
		} else {
			// Generate a random key
			encryptionKey = make([]byte, 32) // AES-256
			if _, err := rand.Read(encryptionKey); err != nil {
				return nil, fmt.Errorf("failed to generate encryption key: %w", err)
			}
		}
	}

	storage := &Storage{
		dataDir:           dataDir,
		backupEnabled:     cfg.BackupEnabled,
		backupInterval:    cfg.BackupInterval,
		encryptionEnabled: cfg.EnableEncryption,
		encryptionKey:     encryptionKey,
		config:            cfg,
	}

	// Create backup directory if backups are enabled
	if storage.backupEnabled {
		backupDir := filepath.Join(dataDir, "backups")
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create backup directory: %w", err)
		}
	}

	return storage, nil
}

// getFilePath returns the file path for the given date
func (s *Storage) getFilePath(date time.Time) string {
	fileName := fmt.Sprintf("sessions_%s.json", date.Format("2006-01-02"))
	return filepath.Join(s.dataDir, fileName)
}

// getBackupPath returns the path for a backup file
func (s *Storage) getBackupPath(date time.Time, timestamp time.Time) string {
	fileName := fmt.Sprintf("sessions_%s_backup_%s.json",
		date.Format("2006-01-02"),
		timestamp.Format("2006-01-02_150405"))
	return filepath.Join(s.dataDir, "backups", fileName)
}

// encrypt encrypts the given data using AES-GCM
func (s *Storage) encrypt(data []byte) ([]byte, error) {
	if !s.encryptionEnabled {
		return data, nil
	}

	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Never use the same nonce for two encryptions with the same key
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Encrypt and authenticate data
	ciphertext := aesgcm.Seal(nil, nonce, data, nil)

	// Prepend nonce to ciphertext
	result := make([]byte, len(nonce)+len(ciphertext))
	copy(result, nonce)
	copy(result[len(nonce):], ciphertext)

	return result, nil
}

// decrypt decrypts the given data using AES-GCM
func (s *Storage) decrypt(data []byte) ([]byte, error) {
	if !s.encryptionEnabled {
		return data, nil
	}

	if len(data) < 13 { // Nonce + at least 1 byte
		return nil, fmt.Errorf("invalid encrypted data: too short")
	}

	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce from beginning of data
	nonce := data[:12]
	ciphertext := data[12:]

	// Decrypt data
	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return plaintext, nil
}

// createBackup creates a backup of the given file
func (s *Storage) createBackup(filePath string, date time.Time) error {
	if !s.backupEnabled {
		return nil
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // Nothing to backup
	}

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file for backup: %w", err)
	}

	// Create backup file
	backupPath := s.getBackupPath(date, time.Now())
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	return nil
}

// SaveDailySessions saves daily sessions to disk
func (s *Storage) SaveDailySessions(sessions *models.DailySessions) error {
	// Add schema version
	sessionsWithSchema := struct {
		SchemaVersion int `json:"schema_version"`
		*models.DailySessions
	}{
		SchemaVersion: config.GetSchemaVersion(),
		DailySessions: sessions,
	}

	// Marshal the data
	data, err := json.MarshalIndent(sessionsWithSchema, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sessions: %w", err)
	}

	// Create a backup before saving (if enabled)
	filePath := s.getFilePath(sessions.Date)
	if err := s.createBackup(filePath, sessions.Date); err != nil {
		// Log error but continue with save
		fmt.Fprintf(os.Stderr, "Warning: failed to create backup: %v\n", err)
	}

	// Encrypt if enabled
	if s.encryptionEnabled {
		data, err = s.encrypt(data)
		if err != nil {
			return fmt.Errorf("failed to encrypt sessions: %w", err)
		}
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write sessions file: %w", err)
	}

	return nil
}

// LoadDailySessions loads daily sessions from disk
func (s *Storage) LoadDailySessions(date time.Time) (*models.DailySessions, error) {
	filePath := s.getFilePath(date)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Return empty sessions for the date
		return &models.DailySessions{
			Date:     date.Truncate(24 * time.Hour),
			Sessions: []*models.Session{},
		}, nil
	}

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read sessions file: %w", err)
	}

	// Decrypt if enabled
	if s.encryptionEnabled {
		data, err = s.decrypt(data)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt sessions: %w", err)
		}
	}

	// Parse the data with schema versioning
	var sessionsWithSchema struct {
		SchemaVersion int `json:"schema_version"`
		models.DailySessions
	}

	if err := json.Unmarshal(data, &sessionsWithSchema); err != nil {
		// Try parsing as old format without schema version
		var oldSessions models.DailySessions
		if innerErr := json.Unmarshal(data, &oldSessions); innerErr != nil {
			return nil, fmt.Errorf("failed to unmarshal sessions: %w", err)
		}

		// Successfully parsed as old format
		return &oldSessions, nil
	}

	// Check if migration is needed
	if sessionsWithSchema.SchemaVersion < config.GetSchemaVersion() {
		// Migrate data to current schema
		migratedSessions, err := s.migrateSchema(
			sessionsWithSchema.SchemaVersion,
			&sessionsWithSchema.DailySessions,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to migrate sessions: %w", err)
		}
		return migratedSessions, nil
	}

	return &sessionsWithSchema.DailySessions, nil
}

// migrateSchema upgrades data from an older schema to the current one
func (s *Storage) migrateSchema(oldVersion int, sessions *models.DailySessions) (*models.DailySessions, error) {
	// For now we don't have migrations, but this provides the framework for adding them
	// as the schema evolves in future versions

	// Migrate schema: add session IDs if they don't exist
	for _, session := range sessions.Sessions {
		if session.ID == "" {
			// Generate a unique ID for the session
			uniqueID := fmt.Sprintf("sess_%d_%d", session.Start.StartTime.UnixNano(), time.Now().UnixNano())
			session.ID = uniqueID
		}
	}

	return sessions, nil
}

// GetDateRange returns a range of dates for stats calculation
func (s *Storage) GetDateRange(rangeType string) (time.Time, time.Time, error) {
	now := time.Now()
	today := now.Truncate(24 * time.Hour)

	switch rangeType {
	case "day":
		return today, today, nil
	case "week":
		// Get the start of the week (Monday)
		weekday := int(now.Weekday())
		if weekday == 0 { // Sunday
			weekday = 7
		}
		startDate := today.AddDate(0, 0, -(weekday - 1))
		return startDate, today, nil
	case "month":
		startDate := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())
		return startDate, today, nil
	case "quarter":
		monthsToSubtract := (int(today.Month()) - 1) % 3
		startDate := time.Date(today.Year(), today.Month()-time.Month(monthsToSubtract), 1, 0, 0, 0, 0, today.Location())
		return startDate, today, nil
	case "year":
		startDate := time.Date(today.Year(), 1, 1, 0, 0, 0, 0, today.Location())
		return startDate, today, nil
	case "all":
		availableDays, err := s.ListAvailableDays()
		if err != nil || len(availableDays) == 0 {
			return today, today, nil
		}
		// Find the earliest date
		earliest := availableDays[0]
		for _, day := range availableDays {
			if day.Before(earliest) {
				earliest = day
			}
		}
		return earliest, today, nil
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("invalid range type: %s", rangeType)
	}
}

// GetStats returns the statistics for the given date range
func (s *Storage) GetStats(rangeType string) (time.Duration, time.Duration, int, error) {
	startDate, endDate, err := s.GetDateRange(rangeType)
	if err != nil {
		return 0, 0, 0, err
	}

	var totalWork, totalInterruption time.Duration
	var totalInterruptionCount int

	// Iterate through each day in the range
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		sessions, err := s.LoadDailySessions(d)
		if err != nil {
			continue // Skip days with errors
		}

		workDuration, interruptionDuration, interruptionCount := sessions.GetStats()
		totalWork += workDuration
		totalInterruption += interruptionDuration
		totalInterruptionCount += interruptionCount
	}

	return totalWork, totalInterruption, totalInterruptionCount, nil
}

// GetDetailedStats returns more detailed statistics for analysis
func (s *Storage) GetDetailedStats(rangeType string) (*models.DetailedStats, error) {
	startDate, endDate, err := s.GetDateRange(rangeType)
	if err != nil {
		return nil, err
	}

	stats := &models.DetailedStats{
		StartDate:                 startDate,
		EndDate:                   endDate,
		TotalWorkDuration:         0,
		TotalInterruptions:        0,
		InterruptionsByTag:        make(map[models.InterruptionTag]int),
		InterruptionDurationByTag: make(map[models.InterruptionTag]time.Duration),
		DailyWorkDurations:        make(map[string]time.Duration),
		HourlyProductivity:        make(map[int]time.Duration),
		LongestSession:            0,
		AverageSessionTime:        0,
		TotalSessions:             0,
	}

	var sessionDurations []time.Duration
	var totalDuration time.Duration

	// Iterate through each day in the range
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dailySessions, err := s.LoadDailySessions(d)
		if err != nil {
			continue // Skip days with errors
		}

		workDuration, _, _ := dailySessions.GetStats()
		stats.DailyWorkDurations[d.Format("2006-01-02")] = workDuration
		stats.TotalWorkDuration += workDuration

		// Process each session
		for _, session := range dailySessions.Sessions {
			if session.Start != nil && session.End != nil {
				sessionDuration := session.End.StartTime.Sub(session.Start.StartTime)

				// Calculate pure work time (excluding interruptions)
				interruptionTime := time.Duration(0)
				for i := 0; i < len(session.Interruptions); i += 2 {
					if i+1 < len(session.Interruptions) {
						interrupt := session.Interruptions[i]
						returnEntry := session.Interruptions[i+1]

						interruptDuration := returnEntry.StartTime.Sub(interrupt.StartTime)
						interruptionTime += interruptDuration

						// Track interruption stats by tag
						tag := interrupt.Tag
						if tag == "" {
							tag = models.TagOther
						}

						stats.InterruptionsByTag[tag]++
						stats.InterruptionDurationByTag[tag] += interruptDuration
						stats.TotalInterruptions++
					}
				}

				pureWorkTime := sessionDuration - interruptionTime

				// Update session stats
				sessionDurations = append(sessionDurations, pureWorkTime)
				totalDuration += pureWorkTime
				stats.TotalSessions++

				if pureWorkTime > stats.LongestSession {
					stats.LongestSession = pureWorkTime
				}

				// Track productivity by hour
				hour := session.Start.StartTime.Hour()
				stats.HourlyProductivity[hour] += pureWorkTime
			}
		}
	}

	// Calculate average session time
	if stats.TotalSessions > 0 {
		stats.AverageSessionTime = totalDuration / time.Duration(stats.TotalSessions)
	}

	return stats, nil
}

// ExportData exports all data to a single JSON file
func (s *Storage) ExportData(outputPath string) error {
	days, err := s.ListAvailableDays()
	if err != nil {
		return fmt.Errorf("failed to list available days: %w", err)
	}

	allData := make(map[string]*models.DailySessions)
	for _, day := range days {
		sessions, err := s.LoadDailySessions(day)
		if err != nil {
			return fmt.Errorf("failed to load sessions for %s: %w", day.Format("2006-01-02"), err)
		}

		allData[day.Format("2006-01-02")] = sessions
	}

	// Marshal the data
	data, err := json.MarshalIndent(allData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal export data: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	return nil
}

// ImportData imports data from a JSON file
func (s *Storage) ImportData(inputPath string, overwrite bool) error {
	// Read the file
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}

	// Parse the data
	var allData map[string]*models.DailySessions
	if err := json.Unmarshal(data, &allData); err != nil {
		return fmt.Errorf("failed to unmarshal import data: %w", err)
	}

	// Import each day's sessions
	for dateStr, sessions := range allData {
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return fmt.Errorf("invalid date format in import: %s", dateStr)
		}

		// If not overwriting, check if file exists
		if !overwrite {
			filePath := s.getFilePath(date)
			if _, err := os.Stat(filePath); err == nil {
				continue // Skip existing files
			}
		}

		// Save the sessions
		sessions.Date = date // Ensure date is set correctly
		if err := s.SaveDailySessions(sessions); err != nil {
			return fmt.Errorf("failed to save imported sessions for %s: %w", dateStr, err)
		}
	}

	return nil
}

// ListAvailableDays returns a list of days that have tracking data
func (s *Storage) ListAvailableDays() ([]time.Time, error) {
	files, err := os.ReadDir(s.dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read data directory: %w", err)
	}

	var days []time.Time
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Parse date from filename (sessions_2025-03-08.json)
		var year, month, day int
		_, err := fmt.Sscanf(file.Name(), "sessions_%d-%d-%d.json", &year, &month, &day)
		if err != nil {
			continue
		}

		date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
		days = append(days, date)
	}

	return days, nil
}

// MergeSessions merges two sessions into one
func (s *Storage) MergeSessions(date time.Time, session1Index, session2Index int) error {
	sessions, err := s.LoadDailySessions(date)
	if err != nil {
		return fmt.Errorf("failed to load sessions: %w", err)
	}

	// Validate indices
	if session1Index < 0 || session1Index >= len(sessions.Sessions) ||
		session2Index < 0 || session2Index >= len(sessions.Sessions) ||
		session1Index == session2Index {
		return fmt.Errorf("invalid session indices")
	}

	// Ensure session1 comes before session2
	if sessions.Sessions[session1Index].Start.StartTime.After(
		sessions.Sessions[session2Index].Start.StartTime) {
		session1Index, session2Index = session2Index, session1Index
	}

	// Get the sessions
	session1 := sessions.Sessions[session1Index]
	session2 := sessions.Sessions[session2Index]

	// Create merged session with a unique ID
	now := time.Now()
	mergedSessionID := fmt.Sprintf("merged_%d", now.UnixNano())

	mergedSession := &models.Session{
		ID:            mergedSessionID,
		Start:         session1.Start,
		End:           session2.End,
		Interruptions: append(session1.Interruptions, session2.Interruptions...),
		SubSessions:   append(session1.SubSessions, session2.SubSessions...),
	}

	// Add an interruption between the sessions if they don't overlap
	if session1.End != nil && session2.Start != nil &&
		session1.End.StartTime.Before(session2.Start.StartTime) {

		// Create interruption entry between sessions
		interruptEntry := models.NewInterruptionEntry("Auto-created gap between merged sessions", models.TagOther)
		interruptEntry.StartTime = session1.End.StartTime

		returnEntry := models.NewTimeEntry(models.EntryTypeReturn, "")
		returnEntry.StartTime = session2.Start.StartTime

		// Add to merged session
		mergedSession.Interruptions = append(
			mergedSession.Interruptions,
			interruptEntry,
			returnEntry,
		)
	}

	// Remove the original sessions
	if session1Index < session2Index {
		sessions.Sessions = append(sessions.Sessions[:session2Index], sessions.Sessions[session2Index+1:]...)
		sessions.Sessions = append(sessions.Sessions[:session1Index], sessions.Sessions[session1Index+1:]...)
	} else {
		sessions.Sessions = append(sessions.Sessions[:session1Index], sessions.Sessions[session1Index+1:]...)
		sessions.Sessions = append(sessions.Sessions[:session2Index], sessions.Sessions[session2Index+1:]...)
	}

	// Add the merged session
	sessions.Sessions = append(sessions.Sessions, mergedSession)

	// Save the changes
	return s.SaveDailySessions(sessions)
}

// SecureDelete permanently deletes a session
func (s *Storage) SecureDelete(date time.Time, sessionIndex int) error {
	sessions, err := s.LoadDailySessions(date)
	if err != nil {
		return fmt.Errorf("failed to load sessions: %w", err)
	}

	// Validate index
	if sessionIndex < 0 || sessionIndex >= len(sessions.Sessions) {
		return fmt.Errorf("invalid session index")
	}

	// Remove the session
	sessions.Sessions = append(sessions.Sessions[:sessionIndex], sessions.Sessions[sessionIndex+1:]...)

	// Save the changes
	return s.SaveDailySessions(sessions)
}

// CreateBackupArchive creates a complete backup of all data
func (s *Storage) CreateBackupArchive(outputPath string) error {
	// For simplicity, this is just a direct copy of the export functionality
	// In a production environment, you might want to use tar/zip compression
	return s.ExportData(outputPath)
}
