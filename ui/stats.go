package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tcell "github.com/gdamore/tcell/v2"
	"github.com/lukaszraczylo/interruption-tracker/models"
	"github.com/rivo/tview"
)

// generateTimelineChart creates a text-based timeline chart for a 24-hour period
func (ui *TimerUI) generateTimelineChart(sessions []*models.Session) string {
	// Get the start of the day (midnight)
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Each hour will have 6 slots (10 min each)
	const intervalsPerHour = 6
	const totalHours = 24
	const totalSlots = totalHours * intervalsPerHour

	// Build activity map: 0 = none, 1 = working, 2 = interrupted, 3 = recovery
	activities := make([]int, totalSlots)

	// Process all sessions to fill activity map
	for _, session := range sessions {
		if session.Start == nil {
			continue
		}

		// Calculate start and end slots
		startTime := session.Start.StartTime

		// If session started before current day, set startTime to beginning of day
		if startTime.Before(startOfDay) {
			startTime = startOfDay
		}

		var endTime time.Time
		if session.End != nil {
			endTime = session.End.StartTime
		} else {
			endTime = time.Now()
		}

		// For timeline display purposes only, cap at end of current day
		displayEndTime := endTime
		if endTime.After(startOfDay.Add(24 * time.Hour)) {
			displayEndTime = startOfDay.Add(24 * time.Hour)
		}

		startSlot := int(startTime.Sub(startOfDay).Minutes()) / (60 / intervalsPerHour)
		endSlot := int(displayEndTime.Sub(startOfDay).Minutes()) / (60 / intervalsPerHour)

		if startSlot < 0 {
			startSlot = 0
		}
		if endSlot >= totalSlots {
			endSlot = totalSlots - 1
		}

		// Mark working periods
		for i := startSlot; i <= endSlot && i < totalSlots; i++ {
			if activities[i] == 0 { // Don't overwrite interruptions/recovery
				activities[i] = 1 // Working
			}
		}

		// If this session continues past midnight, mark the last slot of the day
		if endTime.After(startOfDay.Add(24*time.Hour)) && endSlot == totalSlots-1 {
			activities[totalSlots-1] = 4 // Special marker for crossing midnight
		}

		// Process interruptions and recovery periods
		for i := 0; i < len(session.Interruptions); i += 2 {
			// Get interruption start time
			interruptStart := session.Interruptions[i].StartTime

			// Handle interruptions that start before today or after today
			// If start is before today but end is today, process the part that falls within today
			var processInterruption bool = true
			if interruptStart.Before(startOfDay) {
				if i+1 < len(session.Interruptions) {
					interruptEnd := session.Interruptions[i+1].StartTime
					if interruptEnd.Before(startOfDay) {
						// Both start and end are before today, skip entirely
						processInterruption = false
					} else {
						// Started yesterday, ended today - adjust start time
						interruptStart = startOfDay
					}
				} else {
					// Started before today, still ongoing - adjust start time
					interruptStart = startOfDay
				}
			} else if interruptStart.After(startOfDay.Add(24 * time.Hour)) {
				// Starts after today, skip entirely
				processInterruption = false
			}

			if !processInterruption {
				continue
			}

			// Calculate start slot for interruption
			interruptStartSlot := int(interruptStart.Sub(startOfDay).Minutes()) / (60 / intervalsPerHour)
			if interruptStartSlot < 0 {
				interruptStartSlot = 0
			}

			// Calculate end slot for interruption
			var interruptEnd time.Time
			if i+1 < len(session.Interruptions) {
				interruptEnd = session.Interruptions[i+1].StartTime
			} else {
				interruptEnd = time.Now() // Still interrupted
			}

			// If interruption ends after today, cap at end of day for display
			if interruptEnd.After(startOfDay.Add(24 * time.Hour)) {
				interruptEnd = startOfDay.Add(24 * time.Hour)
			}

			interruptEndSlot := int(interruptEnd.Sub(startOfDay).Minutes()) / (60 / intervalsPerHour)
			if interruptEndSlot >= totalSlots {
				interruptEndSlot = totalSlots - 1
			}

			// Mark interruption on timeline
			for j := interruptStartSlot; j <= interruptEndSlot && j < totalSlots; j++ {
				activities[j] = 2 // Interrupted
			}

			// Add recovery period after each completed interruption
			// BUT only for exactly 10 minutes (1 slot)
			if i+1 < len(session.Interruptions) {
				// Calculate recovery slots (exactly 1 slot for 10 minutes)
				recoveryStartSlot := interruptEndSlot + 1
				recoveryEndSlot := recoveryStartSlot // Only mark one 10-minute slot

				if recoveryEndSlot < totalSlots {
					// Mark exactly one 10-minute slot as recovery
					activities[recoveryEndSlot] = 3 // Recovery
				}
			}
		}
	}

	// Build the timeline chart
	var chart strings.Builder

	// Title
	chart.WriteString("[yellow]Daily Activity Timeline (24-Hour View)[white]\n\n")

	// Create first timeline row with hour markers embedded
	for i := 0; i < totalHours; i++ {
		// Add the hour marker (2 chars) centered in the 6 dots
		chart.WriteString("[blue]")
		chart.WriteString(fmt.Sprintf("%02d", i))
		chart.WriteString("[white]")

		// Add 4 more dots to complete the 6 dots per hour
		chart.WriteString("····")
	}
	chart.WriteString("\n")

	// Second timeline row with activity indicators
	for i := 0; i < totalHours; i++ {
		// 6 activity slots per hour
		for j := 0; j < intervalsPerHour; j++ {
			slotIndex := (i * intervalsPerHour) + j

			if slotIndex < len(activities) {
				switch activities[slotIndex] {
				case 0:
					chart.WriteString("·") // No activity
				case 1:
					chart.WriteString("[green]█[white]") // Working
				case 2:
					chart.WriteString("[red]█[white]") // Interrupted
				case 3:
					chart.WriteString("[yellow]▒[white]") // Recovery
				case 4:
					chart.WriteString("[blue]→[white]") // Continues past midnight
				}
			} else {
				chart.WriteString("·") // Default to no activity
			}
		}
	}
	chart.WriteString("\n\n")

	// Legend
	chart.WriteString("[green]█[white] Working  [red]█[white] Interrupted [yellow]▒[white] Recovery  [blue]→[white] Continues Past Midnight  · No Activity\n\n")

	return chart.String()
}

// Reference to the tasksTable declared in ui.go

// showStats displays statistics for the selected time range
func (ui *TimerUI) showStats(rangeType string) {
	// Ensure our stats view is scrollable
	ui.statsView.SetScrollable(true)

	// Create the tasks table if it doesn't exist
	if tasksTable == nil {
		tasksTable = tview.NewTable().
			SetBorders(true).
			SetFixed(1, 0).
			SetSelectable(true, false). // Allow selecting rows, not columns
			SetSeparator(tview.Borders.Vertical)
	}

	// Recreate the stats page to ensure it adapts to current terminal size
	ui.pages.RemovePage("stats")
	ui.pages.AddPage("stats", ui.createStatsPage(), true, true)

	// Create visualization pages
	ui.createVisualizationPages()

	// Switch to stats page
	ui.pages.SwitchToPage("stats")

	// Get saved statistics from storage (does not include active session)
	workDuration, interruptionDuration, interruptionCount, err := ui.storage.GetStats(rangeType)
	if err != nil {
		ui.statsView.SetText(fmt.Sprintf("[red]Error getting stats: %v", err))
		return
	}

	// Add active session stats if it exists - important for showing current interruptions!
	if ui.activeSession != nil {
		// Get time range for the active session
		activeWorkDuration, activeInterruptDuration, activeInterruptCount :=
			calculateSessionStats(ui.activeSession)

		// Add the active session stats to our totals
		workDuration += activeWorkDuration
		interruptionDuration += activeInterruptDuration
		interruptionCount += activeInterruptCount
	}

	// Format durations
	totalHours := int(workDuration.Hours())
	totalMinutes := int(workDuration.Minutes()) % 60

	interruptHours := int(interruptionDuration.Hours())
	interruptMinutes := int(interruptionDuration.Minutes()) % 60

	// Calculate efficiency percentage with improved algorithm
	var efficiency float64

	// We'll calculate the total actual session time, including interruptions
	// This handles resumed sessions correctly by only counting each session's duration
	// Now properly handles sessions crossing midnight boundaries
	var totalRawSessionTime time.Duration

	for _, session := range ui.currentDay.Sessions {
		if session.Start == nil {
			continue
		}

		// Determine end time for this session - no day boundaries for calculation
		var sessionEndTime time.Time
		if session.End != nil {
			sessionEndTime = session.End.StartTime
		} else {
			sessionEndTime = time.Now() // Active session
		}

		// Add this session's total duration regardless of day boundaries
		// This ensures sessions crossing midnight are properly counted
		totalRawSessionTime += sessionEndTime.Sub(session.Start.StartTime)
	}

	// Calculate total time as the sum of work + interruption
	totalTime := workDuration + interruptionDuration

	// Make sure we don't divide by zero
	if totalRawSessionTime > 0 {
		// Pure work time divided by total session time
		efficiency = float64(workDuration) / float64(totalRawSessionTime) * 100

		// Cap efficiency at 100%
		if efficiency > 100 {
			efficiency = 100.0
		}
	} else if totalTime > 0 {
		// Fallback calculation
		efficiency = float64(workDuration) / float64(totalTime) * 100
	}

	// Build stats text
	rangeText := ""
	switch rangeType {
	case "day":
		rangeText = "Today"
	case "week":
		rangeText = "This Week"
	case "month":
		rangeText = "This Month"
	}

	statsText := fmt.Sprintf(`[yellow]Statistics for %s:

[green]Total Work Time:[white] %d hours, %d minutes
[red]Total Interruption Time*:[white] %d hours, %d minutes
[yellow]Number of Interruptions:[white] %d
[cyan]Work Efficiency:[white] %.1f%%

[gray]*Includes a 10-minute recovery period after each interruption to account for context switching costs[white]

`,
		rangeText,
		totalHours, totalMinutes,
		interruptHours, interruptMinutes,
		interruptionCount,
		efficiency,
	)

	// Add timeline chart only for day view
	// Add timeline chart only for day view
	if rangeType == "day" {
		// Make a copy of sessions and add active session for chart generation
		sessions := make([]*models.Session, len(ui.currentDay.Sessions))
		copy(sessions, ui.currentDay.Sessions)

		// Add active session to the chart
		if ui.activeSession != nil && !containsSession(sessions, ui.activeSession) {
			sessions = append(sessions, ui.activeSession)
		}

		timelineChart := ui.generateTimelineChart(sessions)
		statsText += timelineChart
	}

	// Get completed sessions based on the selected range
	var completedSessions []*models.Session
	startDate, endDate, _ := ui.storage.GetDateRange(rangeType)

	// Iterate through the date range to collect all completed sessions
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		// Load sessions for each day in the range
		dailySessions, err := ui.storage.LoadDailySessions(d)
		if err != nil {
			continue // Skip days with errors
		}

		// Add completed sessions from this day
		for _, session := range dailySessions.Sessions {
			if session.End != nil {
				completedSessions = append(completedSessions, session)
			}
		}
	}

	// Clear the tasks table before populating it
	tasksTable.Clear()

	// Set header row for tasks table
	headers := []string{"Description", "Duration", "Interruptions", "Work Periods", "Total Time"}
	for i, header := range headers {
		// Add padding to headers
		paddedHeader := "  " + header + "  "
		tasksTable.SetCell(0, i,
			tview.NewTableCell(paddedHeader).
				SetTextColor(tcell.ColorYellow).
				SetAlign(tview.AlignCenter).
				SetSelectable(false))
	}

	if len(completedSessions) > 0 {
		// Sort completed sessions by end time (most recent first)
		sort.Slice(completedSessions, func(i, j int) bool {
			return completedSessions[i].End.StartTime.After(completedSessions[j].End.StartTime)
		})

		// Populate the table with session data
		for i, session := range completedSessions {
			row := i + 1 // Start at row 1 (after header)
			// Get total work duration from all sub-sessions
			workDuration := time.Duration(0)
			totalInterruptions := 0

			// Calculate sub-session stats if they exist
			if len(session.SubSessions) > 0 {
				for _, subSession := range session.SubSessions {
					var subEndTime time.Time

					if subSession.End != nil {
						subEndTime = subSession.End.StartTime
					} else {
						continue // Skip incomplete sub-sessions
					}

					// Calculate this sub-session's work time
					subSessionDuration := subEndTime.Sub(subSession.Start.StartTime)
					subInterruptDuration := time.Duration(0)

					// Calculate interruption time for this sub-session
					for i := 0; i < len(subSession.Interruptions); i += 2 {
						if i+1 < len(subSession.Interruptions) {
							interruptStart := subSession.Interruptions[i].StartTime
							interruptEnd := subSession.Interruptions[i+1].StartTime
							subInterruptDuration += interruptEnd.Sub(interruptStart) + (10 * time.Minute) // include recovery
						}
					}

					// Don't let interruption time exceed total time
					if subInterruptDuration > subSessionDuration {
						subInterruptDuration = subSessionDuration
					}

					// Add pure work time for this sub-session
					workDuration += subSessionDuration - subInterruptDuration

					// Count interruptions in this sub-session
					totalInterruptions += len(subSession.Interruptions) / 2
				}
			} else {
				// Legacy session handling
				duration := session.End.StartTime.Sub(session.Start.StartTime)
				interruptCount := len(session.Interruptions) / 2
				interruptDuration := time.Duration(0)

				for i := 0; i < len(session.Interruptions); i += 2 {
					if i+1 < len(session.Interruptions) {
						interruptStart := session.Interruptions[i].StartTime
						interruptEnd := session.Interruptions[i+1].StartTime
						interruptDuration += interruptEnd.Sub(interruptStart) + (10 * time.Minute) // include recovery
					}
				}

				// Don't let interruption time exceed total time
				if interruptDuration > duration {
					interruptDuration = duration
				}

				workDuration = duration - interruptDuration
				totalInterruptions = interruptCount
			}

			// Format duration
			hours := int(workDuration.Hours())
			minutes := int(workDuration.Minutes()) % 60
			durationStr := fmt.Sprintf("%dh %02dm", hours, minutes)

			// Format description
			description := session.Start.Description

			// Add cells to the table with padding
			tasksTable.SetCell(row, 0, tview.NewTableCell("  "+description+"  "))
			tasksTable.SetCell(row, 1, tview.NewTableCell("  "+durationStr+"  "))
			tasksTable.SetCell(row, 2, tview.NewTableCell("  "+fmt.Sprintf("%d", totalInterruptions)+"  "))

			// Set cells for the additional columns
			workPeriodsStr := fmt.Sprintf("%d", len(session.SubSessions))
			if len(session.SubSessions) == 0 {
				workPeriodsStr = "1" // Legacy sessions count as 1 period
			}

			// Calculate total session time from start to end
			totalTime := session.End.StartTime.Sub(session.Start.StartTime)
			totalHours := int(totalTime.Hours())
			totalMinutes := int(totalTime.Minutes()) % 60
			totalTimeStr := fmt.Sprintf("%dh %02dm", totalHours, totalMinutes)

			tasksTable.SetCell(row, 3, tview.NewTableCell("  "+workPeriodsStr+"  "))
			tasksTable.SetCell(row, 4, tview.NewTableCell("  "+totalTimeStr+"  "))
		}

		// Calculate and set optimal column widths based on content
		calculateTableColumnWidths(tasksTable)
	} else {
		// Add a "No completed tasks" message if there are none
		tasksTable.SetCell(1, 0, tview.NewTableCell("  No completed tasks  ").
			SetSelectable(false).
			SetAlign(tview.AlignCenter).
			SetExpansion(1))
		tasksTable.SetCell(1, 1, tview.NewTableCell("    "))
		tasksTable.SetCell(1, 2, tview.NewTableCell("    "))
		tasksTable.SetCell(1, 3, tview.NewTableCell("    "))
		tasksTable.SetCell(1, 4, tview.NewTableCell("    "))
	}

	// Clear the interruptions table
	interruptionsTable.Clear()

	// Set header row for interruptions table
	interruptHeaders := []string{"Type", "Count", "Interrupt", "Recovery", "Total", "Avg Time"}
	for i, header := range interruptHeaders {
		// Add padding to headers
		paddedHeader := "  " + header + "  "
		interruptionsTable.SetCell(0, i,
			tview.NewTableCell(paddedHeader).
				SetTextColor(tcell.ColorYellow).
				SetAlign(tview.AlignCenter).
				SetSelectable(false))
	}

	// Get interruption tag stats from all days in the range
	var allInterruptionStats []models.InterruptionTagStats
	totalInterruptCount := 0

	// Iterate through the date range to collect all interruption stats
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		// Load sessions for each day in the range
		dailySessions, err := ui.storage.LoadDailySessions(d)
		if err != nil {
			continue // Skip days with errors
		}

		// Get stats for this day
		tagStats := dailySessions.GetInterruptionTagStats()

		// Merge with the overall stats
		for _, stat := range tagStats {
			if stat.Count > 0 {
				// Find matching tag in our running stats
				found := false
				for i, existingStat := range allInterruptionStats {
					if existingStat.Tag == stat.Tag {
						// Update existing stat
						allInterruptionStats[i].Count += stat.Count
						allInterruptionStats[i].TotalTime += stat.TotalTime
						allInterruptionStats[i].RecoveryTime += stat.RecoveryTime
						allInterruptionStats[i].TotalWithRecovery += stat.TotalWithRecovery
						found = true
						break
					}
				}

				// If not found, add it
				if !found {
					allInterruptionStats = append(allInterruptionStats, stat)
				}

				totalInterruptCount += stat.Count
			}
		}
	}

	// Recalculate averages for the aggregated stats
	for i := range allInterruptionStats {
		if allInterruptionStats[i].Count > 0 {
			allInterruptionStats[i].AverageTime = allInterruptionStats[i].TotalTime / time.Duration(allInterruptionStats[i].Count)
		}
	}

	if len(allInterruptionStats) > 0 && totalInterruptCount > 0 {
		// Format and display each tag's statistics
		row := 1
		for _, stat := range allInterruptionStats {
			// Skip tags with no interruptions
			if stat.Count == 0 {
				continue
			}

			// Format the tag name
			tagName := string(stat.Tag)
			if tagName == "" {
				tagName = "other" // Default to "other" if tag is empty
			}

			// Format pure interruption time
			interruptMinutes := int(stat.TotalTime.Minutes())
			interruptHours := interruptMinutes / 60
			interruptMinutes = interruptMinutes % 60

			// Format recovery time
			recoveryMinutes := int(stat.RecoveryTime.Minutes())
			recoveryHours := recoveryMinutes / 60
			recoveryMinutes = recoveryMinutes % 60

			// Format total time (interruption + recovery)
			totalMinutes := int(stat.TotalWithRecovery.Minutes())
			totalHours := totalMinutes / 60
			totalMinutes = totalMinutes % 60

			// Format average time (pure interruption)
			avgMinutes := int(stat.AverageTime.Minutes())
			avgHours := avgMinutes / 60
			avgMinutes = avgMinutes % 60

			// Format strings for display
			interruptTimeStr := fmt.Sprintf("%dh %02dm", interruptHours, interruptMinutes)
			recoveryTimeStr := fmt.Sprintf("%dh %02dm", recoveryHours, recoveryMinutes)
			totalTimeStr := fmt.Sprintf("%dh %02dm", totalHours, totalMinutes)
			avgTimeStr := fmt.Sprintf("%dh %02dm", avgHours, avgMinutes)

			// Add the row to the table with padding
			interruptionsTable.SetCell(row, 0, tview.NewTableCell("  "+tagName+"  "))
			interruptionsTable.SetCell(row, 1, tview.NewTableCell("  "+fmt.Sprintf("%d", stat.Count)+"  "))
			interruptionsTable.SetCell(row, 2, tview.NewTableCell("  "+interruptTimeStr+"  "))
			interruptionsTable.SetCell(row, 3, tview.NewTableCell("  "+recoveryTimeStr+"  "))
			interruptionsTable.SetCell(row, 4, tview.NewTableCell("  "+totalTimeStr+"  "))
			interruptionsTable.SetCell(row, 5, tview.NewTableCell("  "+avgTimeStr+"  "))

			row++
		}

		// Calculate and set optimal column widths based on content
		calculateTableColumnWidths(interruptionsTable)

		statsText += "[gray]Note: A 10-minute recovery period is included after each interruption to account for context switching costs[white]\n\n"
	} else {
		// Add a "No interruptions" message if there are none
		interruptionsTable.SetCell(1, 0, tview.NewTableCell("  No interruptions  ").
			SetSelectable(false).
			SetAlign(tview.AlignCenter).
			SetExpansion(1))
		for i := 1; i < 6; i++ {
			interruptionsTable.SetCell(1, i, tview.NewTableCell("    "))
		}
	}
	ui.statsView.SetText(statsText)
}

// calculateSessionStats computes duration and interruption stats for a session
// Now correctly handles sessions that cross midnight
func calculateSessionStats(session *models.Session) (workDuration, interruptDuration time.Duration, interruptCount int) {
	if session.Start == nil {
		return 0, 0, 0
	}

	// Calculate total session time - no limits on duration for crossing midnight
	var endTime time.Time
	if session.End != nil {
		endTime = session.End.StartTime
	} else {
		endTime = time.Now()
	}

	// Use full duration regardless of day boundaries
	totalDuration := endTime.Sub(session.Start.StartTime)
	interruptionDuration := time.Duration(0)
	interruptionCount := 0

	// Calculate interruption time and count
	for i := 0; i < len(session.Interruptions); i += 2 {
		interruptionCount++

		interruptStart := session.Interruptions[i].StartTime
		var interruptEnd time.Time

		if i+1 < len(session.Interruptions) {
			interruptEnd = session.Interruptions[i+1].StartTime

			// Add exact 10-minute recovery period for each completed interruption
			// instead of marking the whole rest of the session
			interruptionDuration += interruptEnd.Sub(interruptStart) + (10 * time.Minute)
		} else {
			// Interruption still active - no recovery time yet
			interruptEnd = time.Now()
			interruptionDuration += interruptEnd.Sub(interruptStart)
		}
	}

	// Make sure interruption time doesn't exceed total time
	if interruptionDuration > totalDuration {
		interruptionDuration = totalDuration
	}

	// Work duration is total time minus interruption time (including recovery periods)
	workDuration = totalDuration - interruptionDuration

	return workDuration, interruptionDuration, interruptionCount
}


// containsSession checks if a session slice contains a specific session
func containsSession(sessions []*models.Session, target *models.Session) bool {
	for _, s := range sessions {
		if s == target {
			return true
		}
	}
	return false
}
