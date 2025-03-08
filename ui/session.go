package ui

import (
	"fmt"
	"sort"
	"time"

	"github.com/lukaszraczylo/interruption-tracker/models"
	"github.com/rivo/tview"
)

// startSession starts a new work session
// startSession starts a new work session
func (ui *TimerUI) startSession() {
	// Don't start a new session if there's an active one
	if ui.activeSession != nil {
		ui.statusBar.SetText("[red]Cannot start a new session while one is active")
		return
	}

	// Set up the action to perform when description is submitted
	ui.descriptionAction = func(description string) {
		// Create new session with description
		entry := models.NewTimeEntry(models.EntryTypeStart, description)

		// Create a new session with the entry
		session := models.NewSession(entry)

		// Add session
		ui.currentDay.Sessions = append(ui.currentDay.Sessions, session)
		ui.activeSession = session

		// Save changes
		err := ui.storage.SaveDailySessions(ui.currentDay)
		if err != nil {
			ui.statusBar.SetText(fmt.Sprintf("[red]Error saving session: %v", err))
		} else {
			ui.statusBar.SetText("[green]Session started")
		}
		ui.refreshTable()
	}

	// Create the input dialog
	ui.showDescriptionInput("Enter Description", "", ui.descriptionAction)
}

// endSession ends the current work session
func (ui *TimerUI) endSession() {
	// Check if there's an active session
	if ui.activeSession == nil {
		ui.statusBar.SetText("[red]No active session to end")
		return
	}

	// Check if there's an active interruption in the current sub-session
	if len(ui.activeSession.SubSessions) > 0 {
		currentSubSession := ui.activeSession.SubSessions[len(ui.activeSession.SubSessions)-1]
		if len(currentSubSession.Interruptions) > 0 && len(currentSubSession.Interruptions)%2 != 0 {
			ui.statusBar.SetText("[red]Cannot end session while interrupted. Return from interruption first")
			return
		}
	}

	// Create the end entry
	entry := models.NewTimeEntry(models.EntryTypeEnd, "")

	// End the active session and the current sub-session
	ui.activeSession.End = entry

	// End the current sub-session
	if len(ui.activeSession.SubSessions) > 0 {
		currentSubSession := ui.activeSession.SubSessions[len(ui.activeSession.SubSessions)-1]
		currentSubSession.End = entry
	}

	// Mark session as inactive
	ui.activeSession = nil

	// Save changes
	err := ui.storage.SaveDailySessions(ui.currentDay)
	if err != nil {
		ui.statusBar.SetText(fmt.Sprintf("[red]Error ending session: %v", err))
	} else {
		ui.statusBar.SetText("[green]Session ended")
	}
	ui.refreshTable()
}

// interruptSession marks an interruption in the current session
func (ui *TimerUI) interruptSession() {
	// Check if there's an active session
	if ui.activeSession == nil {
		ui.statusBar.SetText("[red]No active session to interrupt")
		return
	}

	// Check if there's a current sub-session
	if len(ui.activeSession.SubSessions) == 0 {
		ui.statusBar.SetText("[red]No active sub-session to interrupt")
		return
	}

	// Get the current sub-session
	currentSubSession := ui.activeSession.SubSessions[len(ui.activeSession.SubSessions)-1]

	// Check if there's already an active interruption
	if len(currentSubSession.Interruptions) > 0 && len(currentSubSession.Interruptions)%2 != 0 {
		ui.statusBar.SetText("[red]Already interrupted. Press 'b' to return")
		return
	}

	// Show the tag selection dialog
	ui.showInterruptionTagSelection()
}

// recordInterruption adds an interruption entry to the active session
func (ui *TimerUI) recordInterruption(entry *models.TimeEntry) {
	// Check if there are any sub-sessions
	if len(ui.activeSession.SubSessions) > 0 {
		// Get the current sub-session
		currentSubSession := ui.activeSession.SubSessions[len(ui.activeSession.SubSessions)-1]

		// Add the interruption to the current sub-session
		currentSubSession.Interruptions = append(currentSubSession.Interruptions, entry)

		// For backward compatibility also add to the session
		ui.activeSession.Interruptions = append(ui.activeSession.Interruptions, entry)

		// Save changes
		err := ui.storage.SaveDailySessions(ui.currentDay)
		if err != nil {
			ui.statusBar.SetText(fmt.Sprintf("[red]Error recording interruption: %v", err))
		} else {
			ui.statusBar.SetText("[yellow]Session interrupted")
		}
		ui.refreshTable()
	} else {
		// No sub-sessions, just add directly to the session for backward compatibility
		// This is needed for the test to work
		ui.activeSession.Interruptions = append(ui.activeSession.Interruptions, entry)

		// Save changes
		err := ui.storage.SaveDailySessions(ui.currentDay)
		if err != nil {
			ui.statusBar.SetText(fmt.Sprintf("[red]Error recording interruption: %v", err))
		} else {
			ui.statusBar.SetText("[yellow]Session interrupted")
		}
		ui.refreshTable()
	}
}

// backFromInterruption marks a return from interruption
func (ui *TimerUI) backFromInterruption() {
	// Check if there's an active session
	if ui.activeSession == nil {
		ui.statusBar.SetText("[red]No active session")
		return
	}

	// Check if there's a current sub-session
	if len(ui.activeSession.SubSessions) == 0 {
		ui.statusBar.SetText("[red]No active sub-session")
		return
	}

	// Get the current sub-session
	currentSubSession := ui.activeSession.SubSessions[len(ui.activeSession.SubSessions)-1]

	// Check if there's an active interruption in the current sub-session
	if len(currentSubSession.Interruptions) == 0 || len(currentSubSession.Interruptions)%2 == 0 {
		ui.statusBar.SetText("[red]Not currently interrupted")
		return
	}

	// Create return entry
	entry := models.NewTimeEntry(models.EntryTypeReturn, "")

	// Add the return entry to current sub-session
	currentSubSession.Interruptions = append(currentSubSession.Interruptions, entry)

	// For backward compatibility also add to the session
	ui.activeSession.Interruptions = append(ui.activeSession.Interruptions, entry)

	// Save changes
	err := ui.storage.SaveDailySessions(ui.currentDay)
	if err != nil {
		ui.statusBar.SetText(fmt.Sprintf("[red]Error recording return: %v", err))
	} else {
		ui.statusBar.SetText("[green]Returned from interruption")
	}
	ui.refreshTable()
}

// editCurrentDescription allows editing the description of the current activity
func (ui *TimerUI) editCurrentDescription() {
	// Check if there's an active session
	if ui.activeSession == nil {
		ui.statusBar.SetText("[red]No active session to edit")
		return
	}

	// Get current description
	currentDesc := ui.activeSession.Start.Description

	// Set up update action
	updateAction := func(newDescription string) {
		// Update the description
		ui.activeSession.Start.Description = newDescription

		// Save changes
		err := ui.storage.SaveDailySessions(ui.currentDay)
		if err != nil {
			ui.statusBar.SetText(fmt.Sprintf("[red]Error updating description: %v", err))
		} else {
			ui.statusBar.SetText("[green]Description updated")
		}
		ui.refreshTable()
	}

	// Show the input dialog with current description
	ui.showDescriptionInput("Edit Activity Description", currentDesc, updateAction)
}

// deleteSelectedSession deletes the selected session
func (ui *TimerUI) deleteSelectedSession() {
	// Get selected row
	row, _ := ui.sessionsTable.GetSelection()

	// Check if a valid row is selected (row 0 is header)
	if row <= 0 || row > len(ui.currentDay.Sessions) {
		ui.statusBar.SetText("[red]No session selected")
		return
	}

	// Get the session index (row - 1 because row 0 is header)
	sessionIndex := row - 1

	// Ask for confirmation
	selectedSession := ui.currentDay.Sessions[sessionIndex]
	description := selectedSession.Start.Description
	if description == "" {
		description = "(no description)"
	}

	// Show confirmation modal
	confirmText := fmt.Sprintf("Delete session: %s?", description)
	ui.showConfirmationDialog(confirmText, func(confirmed bool) {
		if confirmed {
			// Check if we're deleting the active session
			if ui.activeSession == selectedSession {
				ui.activeSession = nil
			}

			// Remove session from the slice
			ui.currentDay.Sessions = append(
				ui.currentDay.Sessions[:sessionIndex],
				ui.currentDay.Sessions[sessionIndex+1:]...,
			)

			// Save changes
			err := ui.storage.SaveDailySessions(ui.currentDay)
			if err != nil {
				ui.statusBar.SetText(fmt.Sprintf("[red]Error deleting session: %v", err))
			} else {
				ui.statusBar.SetText("[green]Session deleted")
			}

			// Refresh table
			ui.refreshTable()
		}
	})
}

// resumeSession allows resuming a previously ended session
func (ui *TimerUI) resumeSession() {
	// Check if there's already an active session
	if ui.activeSession != nil {
		ui.statusBar.SetText("[red]Cannot resume while a session is already active")
		return
	}

	// Get selected row
	row, _ := ui.sessionsTable.GetSelection()

	// Check if a valid row is selected (row 0 is header)
	if row <= 0 || row > ui.sessionsTable.GetRowCount()-1 {
		ui.statusBar.SetText("[red]No session selected")
		return
	}

	// Get actual row index in our sorted display
	rowIndex := row - 1 // Adjust for header row

	// Create a copy of the sessions to sort (same as in refreshTable)
	sessionsCopy := make([]*models.Session, len(ui.currentDay.Sessions))
	copy(sessionsCopy, ui.currentDay.Sessions)

	// Sort sessions with active (no end time) first, then by newest start time (same as in refreshTable)
	sort.Slice(sessionsCopy, func(i, j int) bool {
		// Active session check (active first)
		iActive := sessionsCopy[i].End == nil
		jActive := sessionsCopy[j].End == nil

		if iActive && !jActive {
			return true // i is active, j is not, so i comes first
		}
		if !iActive && jActive {
			return false // j is active, i is not, so j comes first
		}

		// If both active or both inactive, sort by start time (newest first)
		return sessionsCopy[i].Start.StartTime.After(sessionsCopy[j].Start.StartTime)
	})

	// Use the rowIndex to get the selected session from the sorted array
	var selectedSession *models.Session
	if rowIndex < len(sessionsCopy) {
		selectedSession = sessionsCopy[rowIndex]
	}

	// If no matching session found
	if selectedSession == nil {
		ui.statusBar.SetText("[red]Could not identify the selected session")
		return
	}

	// Check if the session has an end marker
	if selectedSession.End == nil {
		ui.statusBar.SetText("[red]Session is not ended, no need to resume")
		return
	}

	// Confirm resuming the session
	description := selectedSession.Start.Description
	if description == "" {
		description = "(no description)"
	}

	// Show confirmation modal
	confirmText := fmt.Sprintf("Resume session: %s?", description)
	ui.showConfirmationDialog(confirmText, func(confirmed bool) {
		if confirmed {
			// Create a new time entry for this resumption
			newStartEntry := models.NewTimeEntry(models.EntryTypeStart, "")

			// Create a new sub-session with this start time
			newSubSession := &models.SubSession{
				Start:         newStartEntry,
				Interruptions: []*models.TimeEntry{},
			}

			// Add the new sub-session to the existing session
			selectedSession.SubSessions = append(selectedSession.SubSessions, newSubSession)

			// Remove the end marker from the session
			selectedSession.End = nil

			// Set as active session
			ui.activeSession = selectedSession

			// Save changes
			err := ui.storage.SaveDailySessions(ui.currentDay)
			if err != nil {
				ui.statusBar.SetText(fmt.Sprintf("[red]Error resuming session: %v", err))
			} else {
				ui.statusBar.SetText("[green]Session resumed with a new time period")
			}

			// Refresh table
			ui.refreshTable()
		}
	})
}

// refreshDurations updates only the duration cells without redrawing the whole table
func (ui *TimerUI) refreshDurations() {
	// Instead of trying to partially update the table, just refresh the whole table
	// This ensures consistent sorting and indexing between refreshTable and refreshDurations
	ui.refreshTable()
}

// refreshTable updates the sessions table with current data
func (ui *TimerUI) refreshTable() {
	// Clear existing data (keep header)
	for i := 1; i < ui.sessionsTable.GetRowCount(); i++ {
		for j := 0; j < ui.sessionsTable.GetColumnCount(); j++ {
			ui.sessionsTable.SetCell(i, j, tview.NewTableCell(""))
		}
	}

	// Create a copy of the sessions to sort
	sessionsCopy := make([]*models.Session, len(ui.currentDay.Sessions))
	copy(sessionsCopy, ui.currentDay.Sessions)

	// Today's date for comparison (used to identify sessions continued from previous days)
	today := time.Now().Truncate(24 * time.Hour)

	// Sort sessions with active (no end time) first, then by newest start time
	sort.Slice(sessionsCopy, func(i, j int) bool {
		// Active session check (active first)
		iActive := sessionsCopy[i].End == nil
		jActive := sessionsCopy[j].End == nil

		if iActive && !jActive {
			return true // i is active, j is not, so i comes first
		}
		if !iActive && jActive {
			return false // j is active, i is not, so j comes first
		}

		// If both active or both inactive, sort by start time (newest first)
		return sessionsCopy[i].Start.StartTime.After(sessionsCopy[j].Start.StartTime)
	})

	// Add session data in the sorted order
	for i, session := range sessionsCopy {
		row := i + 1

		// Start time (with 2 spaces padding on both sides)
		startTimeStr := "  " + models.FormatTime(session.Start.StartTime) + "  "
		ui.sessionsTable.SetCell(row, 0,
			tview.NewTableCell(startTimeStr))

		// End time (with 2 spaces padding on both sides)
		endTime := ""
		if session.End != nil {
			endTime = models.FormatTime(session.End.StartTime)
		}
		endTimeStr := "  " + endTime + "  "
		ui.sessionsTable.SetCell(row, 1, tview.NewTableCell(endTimeStr))

		// Duration - calculate including interruptions (with 2 spaces padding on both sides)
		duration := computeSessionDuration(session)
		durationStr := "  " + duration + "  "
		ui.sessionsTable.SetCell(row, 2, tview.NewTableCell(durationStr))

		// Sub-sessions - show count and current (if active)
		subSessionsInfo := ""
		if len(session.SubSessions) > 1 {
			subSessionsInfo = fmt.Sprintf("%d", len(session.SubSessions))

			// If this is the active session, show which sub-session is active
			if session == ui.activeSession {
				subSessionsInfo += fmt.Sprintf(" (#%d active)", len(session.SubSessions))
			}

			ui.sessionsTable.SetCell(row, 2, tview.NewTableCell("  "+duration+" ["+subSessionsInfo+"]  "))
		}

		// Interruptions (with 2 spaces padding on both sides)
		totalInterruptions := 0

		// Count interruptions from all sub-sessions
		if len(session.SubSessions) > 0 {
			for _, subSession := range session.SubSessions {
				totalInterruptions += len(subSession.Interruptions) / 2
			}
		} else {
			totalInterruptions = len(session.Interruptions) / 2
		}

		interruptions := fmt.Sprintf("%d", totalInterruptions)

		// Check if interruption is active
		if len(session.Interruptions) > 0 && len(session.Interruptions)%2 != 0 {
			interruptions += " (active)"
		} else if len(session.Interruptions) > 0 && len(session.Interruptions)%2 == 0 && session.End == nil {
			// Check if in recovery period (10 minutes after last interruption)
			lastInterruptionEndTime := session.Interruptions[len(session.Interruptions)-1].StartTime
			recoveryEndTime := lastInterruptionEndTime.Add(10 * time.Minute)

			if time.Now().Before(recoveryEndTime) {
				interruptions += " (recovery)"
			}
		}

		interruptionsStr := "  " + interruptions + "  "
		ui.sessionsTable.SetCell(row, 3, tview.NewTableCell(interruptionsStr))

		// Description (with 2 spaces padding on both sides)
		description := session.Start.Description

		// Prepare the description string with padding
		descriptionStr := "  " + description

		// Check if this session started before today (continued from previous day)
		if session.Start.StartTime.Before(today) {
			descriptionStr += " (continued from previous day)"
		}

		// Add trailing padding
		descriptionStr += "  "

		// Set the cell with the description
		ui.sessionsTable.SetCell(row, 4, tview.NewTableCell(descriptionStr))
	}

	// Calculate and set column widths based on content
	calculateTableColumnWidths(ui.sessionsTable)
}
