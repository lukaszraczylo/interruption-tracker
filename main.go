package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lukaszraczylo/interruption-tracker/config"
	"github.com/lukaszraczylo/interruption-tracker/storage"
	"github.com/lukaszraczylo/interruption-tracker/ui"
)

// Command line flags
var (
	configFlag    = flag.String("config", "", "Path to configuration file")
	dataFlag      = flag.String("data", "", "Path to data directory")
	exportFlag    = flag.String("export", "", "Export data to file")
	importFlag    = flag.String("import", "", "Import data from file")
	overwriteFlag = flag.Bool("overwrite", false, "Overwrite existing data on import")
	backupFlag    = flag.String("backup", "", "Create backup archive")
	statsFlag     = flag.String("stats", "", "Display stats (day, week, month, quarter, year, all)")
	versionFlag   = flag.Bool("version", false, "Display version information")
)

// Version information
const (
	AppVersion = "1.1.0"
	AppBuild   = "2025-03-08"
)

func main() {
	// Parse flags
	flag.Parse()

	// Show version and exit
	if *versionFlag {
		fmt.Printf("Interruption Tracker version %s (build %s)\n", AppVersion, AppBuild)
		fmt.Println("Go version: 1.23.5")
		os.Exit(0)
	}

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Error loading configuration: %v\n", err)
		fmt.Fprintln(os.Stderr, "Proceeding with default settings")
	}

	// Initialize storage
	dataDir := cfg.DataDirectory
	if *dataFlag != "" {
		dataDir = *dataFlag
	}
	store, err := storage.NewStorage(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing storage: %v\n", err)
		os.Exit(1)
	}

	// Handle utility operations
	if handled := handleUtilityOperations(store); handled {
		os.Exit(0)
	}

	// Initialize UI
	timerUI, err := ui.NewTimerUI(store)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing UI: %v\n", err)
		os.Exit(1)
	}

	// Run the application
	if err := timerUI.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running application: %v\n", err)
		os.Exit(1)
	}
}

// loadConfig loads the configuration from file or creates default
func loadConfig() (*config.Config, error) {
	if *configFlag != "" {
		// Load from custom config file path
		return config.LoadConfigFromPath(*configFlag)
	}

	return config.LoadConfig()
}

// handleUtilityOperations processes command-line utility operations
// Returns true if an operation was performed and the app should exit
func handleUtilityOperations(store *storage.Storage) bool {
	// Export data
	if *exportFlag != "" {
		exportPath := *exportFlag
		fmt.Printf("Exporting data to %s...\n", exportPath)
		if err := store.ExportData(exportPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error exporting data: %v\n", err)
			return true
		}
		fmt.Println("Export completed successfully.")
		return true
	}

	// Import data
	if *importFlag != "" {
		importPath := *importFlag
		fmt.Printf("Importing data from %s...\n", importPath)
		if err := store.ImportData(importPath, *overwriteFlag); err != nil {
			fmt.Fprintf(os.Stderr, "Error importing data: %v\n", err)
			return true
		}
		fmt.Println("Import completed successfully.")
		return true
	}

	// Create backup archive
	if *backupFlag != "" {
		backupPath := *backupFlag
		fmt.Printf("Creating backup archive at %s...\n", backupPath)
		if err := store.CreateBackupArchive(backupPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating backup: %v\n", err)
			return true
		}
		fmt.Println("Backup created successfully.")
		return true
	}

	// Display stats
	if *statsFlag != "" {
		rangeType := *statsFlag
		displayConsoleStats(store, rangeType)
		return true
	}

	return false
}

// displayConsoleStats shows statistics in the console (non-UI mode)
func displayConsoleStats(store *storage.Storage, rangeType string) {
	// Get basic stats
	workDuration, interruptionDuration, interruptionCount, err := store.GetStats(rangeType)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting stats: %v\n", err)
		return
	}

	// Get date range
	startDate, endDate, _ := store.GetDateRange(rangeType)

	// Display header
	fmt.Printf("Statistics for %s (%s to %s)\n",
		rangeType,
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"))
	fmt.Println(strings.Repeat("-", 50))

	// Display basic metrics
	fmt.Printf("Total work time: %s\n", formatDuration(workDuration))
	fmt.Printf("Total interruptions: %d\n", interruptionCount)
	fmt.Printf("Total interruption time: %s\n", formatDuration(interruptionDuration))

	// Recovery time (10 min per interruption)
	recoveryTime := time.Duration(interruptionCount) * 10 * time.Minute
	fmt.Printf("Estimated recovery time: %s\n", formatDuration(recoveryTime))

	// Total impact
	totalImpact := interruptionDuration + recoveryTime
	fmt.Printf("Total productivity impact: %s\n", formatDuration(totalImpact))

	// Get detailed stats if available
	detailedStats, err := store.GetDetailedStats(rangeType)
	if err == nil && detailedStats != nil {
		// Calculate productivity score
		score := detailedStats.CalculateProductivityScore()
		fmt.Printf("Productivity score: %.1f / 100\n", score)

		// Most productive hour
		if hour, duration := detailedStats.GetMostProductiveHour(); duration > 0 {
			fmt.Printf("Most productive hour: %d:00 (%s of focused work)\n",
				hour, formatDuration(duration))
		}

		// Display interruption breakdown
		if len(detailedStats.InterruptionsByTag) > 0 {
			fmt.Println("\nInterruption breakdown:")
			fmt.Println(strings.Repeat("-", 50))
			fmt.Printf("%-10s %-10s %-15s\n", "Type", "Count", "Duration")

			for tag, count := range detailedStats.InterruptionsByTag {
				duration := detailedStats.InterruptionDurationByTag[tag]
				fmt.Printf("%-10s %-10d %-15s\n",
					string(tag), count, formatDuration(duration))
			}
		}
	}
}

// formatDuration formats a duration in a human-readable format
func formatDuration(d time.Duration) string {
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
