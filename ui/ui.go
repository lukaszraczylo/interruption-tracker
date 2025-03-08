package ui

import (
	"fmt"
	"sort"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/lukaszraczylo/interruption-tracker/models"
	"github.com/lukaszraczylo/interruption-tracker/storage"
	"github.com/rivo/tview"
)

// TimerUI represents the main UI of the application
type TimerUI struct {
	app           *tview.Application
	pages         *tview.Pages
	mainGrid      *tview.Grid
	sessionsTable *tview.Table
	statusBar     *tview.TextView
	inputField    *tview.InputField
	statsView     *tview.TextView

	storage       *storage.Storage
	currentDay    *models.DailySessions
	activeSession *models.Session

	// Action to perform when description is submitted
	descriptionAction func(string)
}

// NewTimerUI creates a new UI instance
func NewTimerUI(storage *storage.Storage) (*TimerUI, error) {
	// Load today's sessions
	today := time.Now().Truncate(24 * time.Hour)
	dailySessions, err := storage.LoadDailySessions(today)
	if err != nil {
		return nil, fmt.Errorf("failed to load daily sessions: %w", err)
	}

	// Create UI instance
	ui := &TimerUI{
		app:        tview.NewApplication(),
		pages:      tview.NewPages(),
		storage:    storage,
		currentDay: dailySessions,
	}

	// Find active session if any
	for _, session := range dailySessions.Sessions {
		if session.End == nil {
			ui.activeSession = session
			break
		}
	}

	// If no active session found in current day, check previous day for active sessions
	if ui.activeSession == nil {
		// Check if there's an active session from the previous day
		previousDay := today.AddDate(0, 0, -1)
		previousSessions, err := storage.LoadDailySessions(previousDay)
		if err == nil { // Ignore errors as previous day may not exist
			var activeSessionFromPreviousDay *models.Session

			// Find any active session from previous day
			for _, session := range previousSessions.Sessions {
				if session.End == nil {
					activeSessionFromPreviousDay = session
					break
				}
			}

			// If an active session exists in the previous day, move it to today
			if activeSessionFromPreviousDay != nil {
				// Add the session to current day's sessions
				ui.currentDay.Sessions = append(ui.currentDay.Sessions, activeSessionFromPreviousDay)
				ui.activeSession = activeSessionFromPreviousDay

				// Save the current day with the moved session
				err := ui.storage.SaveDailySessions(ui.currentDay)
				if err != nil {
					return nil, fmt.Errorf("failed to save session moved from previous day: %w", err)
				}

				// Remove the session from previous day's sessions
				newPreviousSessions := []*models.Session{}
				for _, s := range previousSessions.Sessions {
					if s != activeSessionFromPreviousDay {
						newPreviousSessions = append(newPreviousSessions, s)
					}
				}
				previousSessions.Sessions = newPreviousSessions

				// Save the updated previous day's sessions
				err = ui.storage.SaveDailySessions(previousSessions)
				if err != nil {
					return nil, fmt.Errorf("failed to update previous day after moving session: %w", err)
				}
			}
		}
	}

	// Initialize UI components
	ui.setupUI()

	return ui, nil
}

// setupUI initializes the UI components
func (ui *TimerUI) setupUI() {
	// Create sessions table
	ui.sessionsTable = tview.NewTable().
		SetBorders(true).
		SetFixed(1, 0).
		SetSelectable(true, false). // Allow selecting rows, not columns
		SetSeparator(tview.Borders.Vertical).
		SetSelectedStyle(tcell.Style{}.
			Background(tcell.ColorNavy).
			Foreground(tcell.ColorWhite)) // Apply selection style only to cell content

	// Set header row
	headers := []string{"Start", "End", "Duration", "Interruptions", "Description"}
	for i, header := range headers {
		// Add 2 spaces padding on both sides
		paddedHeader := "  " + header + "  "
		ui.sessionsTable.SetCell(0, i,
			tview.NewTableCell(paddedHeader).
				SetTextColor(tcell.ColorYellow).
				SetAlign(tview.AlignCenter).
				SetSelectable(false))
	}

	// Create status bar
	ui.statusBar = tview.NewTextView().
		SetDynamicColors(true).
		SetText("[yellow]Press (s)tart, (e)nd, (i)nterrupt, (b)ack, (d)elete, (r)ename, (u)ndo end, (v)iew stats, (q)uit")

	// Create input field for descriptions
	ui.inputField = tview.NewInputField().
		SetLabel("Description: ").
		SetFieldWidth(0) // 0 means use all available space
	ui.inputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			text := ui.inputField.GetText()

			if ui.descriptionAction != nil {
				ui.descriptionAction(text)
				ui.descriptionAction = nil
			}

			ui.app.QueueUpdateDraw(func() {
				ui.inputField.SetText("")
				ui.mainGrid.RemoveItem(ui.inputField)
				ui.app.SetFocus(ui.sessionsTable)
			})
		}
	})

	// Create stats view
	ui.statsView = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)

	// Create main grid layout that adapts to terminal size
	ui.mainGrid = tview.NewGrid().
		SetRows(1, 0, 1).
		SetColumns(0).
		SetBorders(false)

	// Add elements to grid
	ui.mainGrid.AddItem(tview.NewTextView().SetText(" Interruption Tracker").SetTextColor(tcell.ColorGreen), 0, 0, 1, 1, 0, 0, false)
	ui.mainGrid.AddItem(ui.sessionsTable, 1, 0, 1, 1, 0, 0, true)
	ui.mainGrid.AddItem(ui.statusBar, 2, 0, 1, 1, 0, 0, false)

	// Create pages for different views
	ui.pages.AddPage("main", ui.mainGrid, true, true)
	ui.pages.AddPage("stats", ui.createStatsPage(), true, false)
}

// tasksTable is a table component for displaying completed tasks
var tasksTable *tview.Table

// interruptionsTable is a table component for displaying interruption statistics
var interruptionsTable *tview.Table

// createStatsPage creates a stats view page that adapts to the terminal size
func (ui *TimerUI) createStatsPage() tview.Primitive {
	// Use a flexible layout with rows for header, stats view, section headers, tables, and footer
	statsGrid := tview.NewGrid().
		SetRows(1, 0, 1, 10, 1, 8, 1). // Main header, stats view, tasks header, tasks table, interruptions header, interruptions table, footer
		SetColumns(0)

	statsHeader := tview.NewTextView().
		SetText(" Statistics").
		SetTextColor(tcell.ColorGreen)

	tasksHeader := tview.NewTextView().
		SetText(" Completed Tasks").
		SetTextColor(tcell.ColorYellow)

	interruptionsHeader := tview.NewTextView().
		SetText(" Interruption Breakdown").
		SetTextColor(tcell.ColorYellow)

	statsFooter := tview.NewTextView().
		SetText(" Press (d)ay, (w)eek, (m)onth, (p)roductivity, (t)rends, (i)nterruptions, (b)ack, (q)uit").
		SetTextColor(tcell.ColorYellow)

	// Enable scrolling for the stats view
	ui.statsView.SetScrollable(true)

	// Create the tasks table if it doesn't exist
	if tasksTable == nil {
		tasksTable = tview.NewTable().
			SetBorders(true).
			SetFixed(1, 0).
			SetSelectable(false, false). // Disable selection
			SetSeparator(tview.Borders.Vertical).
			SetSelectedStyle(tcell.Style{}.
				Background(tcell.ColorNavy).
				Foreground(tcell.ColorWhite)) // Apply selection style only to cell content
	}

	// Create the interruptions table if it doesn't exist
	if interruptionsTable == nil {
		interruptionsTable = tview.NewTable().
			SetBorders(true).
			SetFixed(1, 0).
			SetSelectable(false, false). // Disable selection
			SetSeparator(tview.Borders.Vertical).
			SetSelectedStyle(tcell.Style{}.
				Background(tcell.ColorNavy).
				Foreground(tcell.ColorWhite)) // Apply selection style only to cell content
	}

	// Set header row for tasks table
	taskHeaders := []string{"Description", "Duration", "Interruptions", "Start Time", "End Time"}
	for i, header := range taskHeaders {
		// Add 2 spaces padding on both sides
		paddedHeader := "  " + header + "  "
		tasksTable.SetCell(0, i,
			tview.NewTableCell(paddedHeader).
				SetTextColor(tcell.ColorYellow).
				SetAlign(tview.AlignCenter).
				SetSelectable(false))
	}

	// Set header row for interruptions table
	interruptHeaders := []string{"Type", "Count", "Interrupt", "Recovery", "Total", "Avg Time"}
	for i, header := range interruptHeaders {
		// Add 2 spaces padding on both sides
		paddedHeader := "  " + header + "  "
		interruptionsTable.SetCell(0, i,
			tview.NewTableCell(paddedHeader).
				SetTextColor(tcell.ColorYellow).
				SetAlign(tview.AlignCenter).
				SetSelectable(false))
	}

	// Add items to grid
	statsGrid.AddItem(statsHeader, 0, 0, 1, 1, 0, 0, false)
	statsGrid.AddItem(ui.statsView, 1, 0, 1, 1, 0, 0, false)
	statsGrid.AddItem(tasksHeader, 2, 0, 1, 1, 0, 0, false)
	statsGrid.AddItem(tasksTable, 3, 0, 1, 1, 0, 0, false) // No longer focusable
	statsGrid.AddItem(interruptionsHeader, 4, 0, 1, 1, 0, 0, false)
	statsGrid.AddItem(interruptionsTable, 5, 0, 1, 1, 0, 0, false)
	statsGrid.AddItem(statsFooter, 6, 0, 1, 1, 0, 0, false)

	return statsGrid
}

// KeyHandler handles key events, returns true if the key was handled
func (ui *TimerUI) KeyHandler(key *tcell.EventKey) bool {
	// Check current page
	currentPage, _ := ui.pages.GetFrontPage()

	// Don't intercept key events on the input modal
	if currentPage == "input" {
		return false
	}

	// First, try to handle with the extended key handler (for visualizations)
	if ui.extendedKeyHandler(key) {
		return true
	}

	// Handle main page keys
	if currentPage == "main" {
		// Handle special keys first
		if key.Key() == tcell.KeyEnter {
			ui.showSessionDetailsModal()
			return true
		}

		switch key.Rune() {
		case 's', 'S':
			ui.startSession()
			return true
		case 'e', 'E':
			ui.endSession()
			return true
		case 'i', 'I':
			ui.interruptSession()
			return true
		case 'b', 'B':
			ui.backFromInterruption()
			return true
		case 'v', 'V':
			ui.showStats("day")
			return true
		case 'd', 'D':
			ui.deleteSelectedSession()
			return true
		case 'q', 'Q':
			ui.app.Stop()
			return true
		case 'r', 'R':
			ui.editCurrentDescription()
			return true
		case 'u', 'U':
			ui.resumeSession()
			return true
		}
	} else if currentPage == "stats" {
		// Handle stats page keys
		switch key.Rune() {
		case 'd', 'D':
			ui.showStats("day")
			return true
		case 'w', 'W':
			ui.showStats("week")
			return true
		case 'm', 'M':
			ui.showStats("month")
			return true
		case 'q', 'Q':
			// Handle 'q' to quit from stats page
			ui.app.Stop()
			return true
		case 'y', 'Y':
			ui.showStats("year")
			return true
		case 'a', 'A':
			ui.showStats("all")
			return true
		case 'b', 'B':
			ui.pages.SwitchToPage("main")
			return true
		case 'v', 'V':
			ui.pages.SwitchToPage("main")
			return true
		case 'h', 'H':
			// Toggle heatmap view
			ui.pages.SwitchToPage("productivity")
			return true
		}
	}

	return false
}

// Run starts the UI
func (ui *TimerUI) Run() error {
	// Set up a ticker to update durations for active sessions
	ticker := time.NewTicker(1 * time.Second)
	go func() {
		for range ticker.C {
			// Only update if there's an active session
			if ui.activeSession != nil {
				ui.app.QueueUpdateDraw(func() {
					ui.refreshDurations() // Only update durations, not the whole table
				})
			}
		}
	}()

	// Make sure to stop the ticker when the application exits
	defer ticker.Stop()

	// Pre-populate the sessions table
	ui.refreshTable()

	// Set our key handler for the application
	ui.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle Ctrl+C to quit
		if event.Key() == tcell.KeyCtrlC {
			ui.app.Stop()
			return nil
		}

		// Use our key handler
		if ui.KeyHandler(event) {
			return nil
		}

		return event
	})

	// Make sure stats view and table are scrollable
	ui.statsView.SetScrollable(true)
	ui.sessionsTable.SetSelectable(true, false) // Already added, but this ensures row selection

	// Set a function to adjust UI based on screen size before drawing
	ui.app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		width, height := screen.Size()
		if width > 10 {
			// Let our column width calculation function handle most columns
			widths := calculateTableColumnWidths(ui.sessionsTable)

			// Ensure minimum widths for time columns
			if len(widths) >= 5 {
				// Make sure time columns have at least 16 characters width (HH:MM:SS + padding)
				if widths[0] < 16 {
					widths[0] = 16 // Start time
				}
				if widths[1] < 16 {
					widths[1] = 16 // End time
				}

				// Description column gets remaining space with a minimum
				descColWidth := width - widths[0] - widths[1] - widths[2] - widths[3] - 10 // 10 for borders/padding
				if descColWidth < 25 {
					descColWidth = 25 // Minimum width for description
				}
				widths[4] = descColWidth

				// Apply the adjusted widths
				for i, w := range widths {
					if i < ui.sessionsTable.GetColumnCount() {
						for row := 0; row < ui.sessionsTable.GetRowCount(); row++ {
							cell := ui.sessionsTable.GetCell(row, i)
							if cell != nil {
								cell.SetMaxWidth(w)
							}
						}
					}
				}
			}

			// Use the terminal height to adjust grid dimensions
			// The main grid has 3 rows: header, content, footer
			// We want the content to take most of the space
			contentHeight := height - 2 // Reserve 2 lines for header and footer
			if contentHeight < 1 {
				contentHeight = 1 // Minimum height
			}
			ui.mainGrid.SetRows(1, contentHeight, 1)

			// We'll recreate the stats page whenever we switch to it
		}

		// Reset status bar to standard instructions based on current page
		currentPage, _ := ui.pages.GetFrontPage()
		if currentPage == "main" {
			ui.statusBar.SetText("[yellow]Press (s)tart, (e)nd, (i)nterrupt, (b)ack, (d)elete, (r)ename, (u)ndo end, (v)iew stats, (Enter) details, (q)uit")
		} else if currentPage == "stats" {
			ui.statusBar.SetText("[yellow]Press (d)ay, (w)eek, (m)onth, (b)ack, (q)uit")
		}

		return false // Continue with the actual drawing
	})

	// Start the application with mouse support
	ui.app.SetRoot(ui.pages, true).EnableMouse(true)
	return ui.app.Run()
}

// showDescriptionInput displays a dialog for entering or editing a description
func (ui *TimerUI) showDescriptionInput(title, initialValue string, callback func(string)) {
	// Create an input modal
	inputField := tview.NewInputField().
		SetLabel("Description: ").
		SetFieldWidth(40).
		SetText(initialValue)

	// Set done function that handles Enter key
	inputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			description := inputField.GetText()
			ui.pages.RemovePage("input")
			ui.app.SetFocus(ui.sessionsTable)

			if callback != nil {
				callback(description)
			}
		}
	})

	// Create a form to hold the input field and button
	buttonText := "Submit"
	if initialValue != "" {
		buttonText = "Update"
	}

	inputForm := tview.NewForm().
		AddFormItem(inputField).
		AddButton(buttonText, func() {
			description := inputField.GetText()
			ui.pages.RemovePage("input")
			ui.app.SetFocus(ui.sessionsTable)

			if callback != nil {
				callback(description)
			}
		}).
		AddButton("Cancel", func() {
			ui.pages.RemovePage("input")
			ui.app.SetFocus(ui.sessionsTable)
		})

	inputForm.SetBorder(true)
	inputForm.SetTitle(" " + title + " ")
	inputForm.SetTitleAlign(tview.AlignCenter)

	// Create a flex layout for centering the form
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(inputForm, 60, 1, true).
			AddItem(nil, 0, 1, false),
			10, 1, true).
		AddItem(nil, 0, 1, false)

	// Make sure to capture escape key to close the dialog
	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			ui.pages.RemovePage("input")
			ui.app.SetFocus(ui.sessionsTable)
			return nil
		}
		return event
	})

	// Add the input modal as a page
	ui.pages.AddPage("input", flex, true, true)
	ui.app.SetFocus(inputField) // Set focus on the input field directly
}

// showInterruptionTagSelection shows the dialog for selecting interruption tags
func (ui *TimerUI) showInterruptionTagSelection() {
	// Create a tag selection modal
	modal := tview.NewModal().
		SetText("Select interruption type:").
		AddButtons([]string{
			"1. Call",
			"2. Meeting",
			"3. Spouse",
			"4. Other (custom)",
		})

	// Create a map of available tags
	tags := []models.InterruptionTag{
		models.TagCall,
		models.TagMeeting,
		models.TagSpouse,
		models.TagOther,
	}

	// Handle tag selection
	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		ui.pages.RemovePage("tag_select")

		if buttonIndex < 0 {
			// Cancelled
			ui.app.SetFocus(ui.sessionsTable)
			return
		}

		// Custom interruption needs description
		if buttonIndex == 3 { // Other
			ui.showInterruptionDescriptionInput(models.TagOther)
		} else {
			// Create a new interruption with the selected tag and empty description
			entry := models.NewInterruptionEntry("", tags[buttonIndex])
			ui.recordInterruption(entry)
		}
	})

	// Set key handlers for quick number selection
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Convert rune to integer (1-4)
		if event.Key() == tcell.KeyRune {
			num := int(event.Rune() - '0')
			if num >= 1 && num <= 4 {
				ui.pages.RemovePage("tag_select")

				if num == 4 { // Other
					ui.showInterruptionDescriptionInput(models.TagOther)
				} else {
					// Create a new interruption with the selected tag and empty description
					entry := models.NewInterruptionEntry("", tags[num-1])
					ui.recordInterruption(entry)
				}
				return nil
			}
		}

		return event
	})

	// Show the modal
	ui.pages.AddPage("tag_select", modal, true, true)
	ui.app.SetFocus(modal)
}

// showInterruptionDescriptionInput shows a modal for entering interruption description
func (ui *TimerUI) showInterruptionDescriptionInput(tag models.InterruptionTag) {
	// Create an input modal
	inputField := tview.NewInputField().
		SetLabel("Description: ").
		SetFieldWidth(40)

	// Set done function that handles Enter key
	inputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			description := inputField.GetText()
			ui.pages.RemovePage("input")
			ui.app.SetFocus(ui.sessionsTable)

			// Create and record the interruption
			entry := models.NewInterruptionEntry(description, tag)
			ui.recordInterruption(entry)
		}
	})

	// Create a form to hold the input field and button
	inputForm := tview.NewForm().
		AddFormItem(inputField).
		AddButton("Submit", func() {
			description := inputField.GetText()
			ui.pages.RemovePage("input")
			ui.app.SetFocus(ui.sessionsTable)

			// Create and record the interruption
			entry := models.NewInterruptionEntry(description, tag)
			ui.recordInterruption(entry)
		}).
		AddButton("Cancel", func() {
			ui.pages.RemovePage("input")
			ui.app.SetFocus(ui.sessionsTable)
		})

	inputForm.SetBorder(true)
	inputForm.SetTitle(" Enter Interruption Description ")
	inputForm.SetTitleAlign(tview.AlignCenter)

	// Create a flex layout for centering the form
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(inputForm, 60, 1, true).
			AddItem(nil, 0, 1, false),
			10, 1, true).
		AddItem(nil, 0, 1, false)

	// Make sure to capture escape key to close the dialog
	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			ui.pages.RemovePage("input")
			ui.app.SetFocus(ui.sessionsTable)
			return nil
		}
		return event
	})

	// Add the input modal as a page
	ui.pages.AddPage("input", flex, true, true)
	ui.app.SetFocus(inputField) // Set focus on the input field directly
}

// showConfirmationDialog displays a confirmation dialog and calls callback with the result
func (ui *TimerUI) showConfirmationDialog(message string, callback func(bool)) {
	// Create confirmation modal
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"Yes", "No"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			confirmed := buttonIndex == 0
			ui.pages.RemovePage("confirm")
			ui.app.SetFocus(ui.sessionsTable)
			callback(confirmed)
		})

	// Add the confirmation modal as a page
	ui.pages.AddPage("confirm", modal, true, true)
	ui.app.SetFocus(modal)
}

// showSessionDetailsModal displays a modal with detailed information about the selected session
func (ui *TimerUI) showSessionDetailsModal() {
	// Get selected row
	row, _ := ui.sessionsTable.GetSelection()

	// Check if a valid row is selected (row 0 is header)
	if row <= 0 || row > len(ui.currentDay.Sessions) {
		ui.statusBar.SetText("[red]No session selected")
		return
	}

	// Get the session index (row - 1 because row 0 is header)
	rowIndex := row - 1

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

	// Create a flex container for the modal
	modalFlex := tview.NewFlex().
		SetDirection(tview.FlexRow)

	// Add session header information
	headerText := fmt.Sprintf(" Session: %s\n Start: %s\n",
		selectedSession.Start.Description,
		models.FormatTime(selectedSession.Start.StartTime))

	if selectedSession.End != nil {
		headerText += fmt.Sprintf(" End: %s\n", models.FormatTime(selectedSession.End.StartTime))
	} else {
		headerText += " End: [yellow]Active[white]\n"
	}

	headerText += fmt.Sprintf(" Total Duration: %s\n", computeSessionDuration(selectedSession))

	header := tview.NewTextView().
		SetText(headerText).
		SetDynamicColors(true)

	modalFlex.AddItem(header, 5, 0, false)

	// Create a table for sub-sessions
	subSessionsTable := tview.NewTable().
		SetBorders(true).
		SetSeparator(tview.Borders.Vertical).
		SetSelectable(true, false).
		SetSelectedStyle(tcell.Style{}.
			Background(tcell.ColorNavy).
			Foreground(tcell.ColorWhite)) // Apply selection style only to cell content

	// Set header row for sub-sessions table
	headers := []string{"Sub-Session", "Start", "End", "Duration", "Interruptions"}
	for i, header := range headers {
		subSessionsTable.SetCell(0, i,
			tview.NewTableCell(header).
				SetTextColor(tcell.ColorYellow).
				SetAlign(tview.AlignCenter).
				SetSelectable(false))
	}

	// Sort sub-sessions from newest to oldest
	subSessionsCopy := make([]*models.SubSession, len(selectedSession.SubSessions))
	copy(subSessionsCopy, selectedSession.SubSessions)

	sort.Slice(subSessionsCopy, func(i, j int) bool {
		// Active session check (active first)
		iActive := subSessionsCopy[i].End == nil
		jActive := subSessionsCopy[j].End == nil

		if iActive && !jActive {
			return true // i is active, j is not, so i comes first
		}
		if !iActive && jActive {
			return false // j is active, i is not, so j comes first
		}

		// If both active or both inactive, sort by start time (newest first)
		return subSessionsCopy[i].Start.StartTime.After(subSessionsCopy[j].Start.StartTime)
	})

	// Populate sub-sessions table
	for i, subSession := range subSessionsCopy {
		row := i + 1

		// Find original index of this sub-session for displaying
		originalIndex := -1
		for idx, origSubSession := range selectedSession.SubSessions {
			if origSubSession == subSession {
				originalIndex = idx
				break
			}
		}

		// Sub-session number (from original order)
		subSessionsTable.SetCell(row, 0,
			tview.NewTableCell(fmt.Sprintf("#%d", originalIndex+1)).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignCenter))

		// Start time
		subSessionsTable.SetCell(row, 1,
			tview.NewTableCell(models.FormatTime(subSession.Start.StartTime)).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignCenter))

		// End time
		endTimeText := "[yellow]Active[white]"
		if subSession.End != nil {
			endTimeText = models.FormatTime(subSession.End.StartTime)
		}
		subSessionsTable.SetCell(row, 2,
			tview.NewTableCell(endTimeText).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignCenter))

		// Duration
		var duration string
		var startTime = subSession.Start.StartTime
		var endTime time.Time

		if subSession.End != nil {
			endTime = subSession.End.StartTime
		} else {
			endTime = time.Now()
		}

		// Calculate duration excluding interruptions
		totalDuration := endTime.Sub(startTime)
		interruptionDuration := time.Duration(0)

		for i := 0; i < len(subSession.Interruptions); i += 2 {
			interruptStart := subSession.Interruptions[i].StartTime

			var interruptEnd time.Time
			if i+1 < len(subSession.Interruptions) {
				interruptEnd = subSession.Interruptions[i+1].StartTime
			} else {
				interruptEnd = time.Now()
			}

			interruptionDuration += interruptEnd.Sub(interruptStart)
		}

		effectiveDuration := totalDuration - interruptionDuration
		hours := int(effectiveDuration.Hours())
		minutes := int(effectiveDuration.Minutes()) % 60
		seconds := int(effectiveDuration.Seconds()) % 60

		duration = fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)

		subSessionsTable.SetCell(row, 3,
			tview.NewTableCell(duration).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignCenter))

		// Interruptions count
		interruptionsCount := len(subSession.Interruptions) / 2
		if len(subSession.Interruptions)%2 != 0 {
			// There's an active interruption
			interruptionsCount = len(subSession.Interruptions)/2 + 1
		}

		subSessionsTable.SetCell(row, 4,
			tview.NewTableCell(fmt.Sprintf("%d", interruptionsCount)).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignCenter))
	}

	// Calculate column widths for the sub-sessions table
	calculateTableColumnWidths(subSessionsTable)

	// Limit table to show only 4 records at a time (plus header row)
	tableHeight := 5 // header row + 4 content rows
	if subSessionsTable.GetRowCount() < tableHeight {
		tableHeight = subSessionsTable.GetRowCount()
	}

	// Make table scrollable
	modalFlex.AddItem(subSessionsTable, tableHeight, 0, true)

	// Create a text view for interruptions details with a clearly defined height
	interruptionsText := tview.NewTextView().
		SetText("Select a sub-session to view interruption details").
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetScrollable(true)

	modalFlex.AddItem(interruptionsText, 10, 0, false)

	// Handle selection change in sub-sessions table to show interruption details
	subSessionsTable.SetSelectedFunc(func(row, column int) {
		if row == 0 { // Header row
			return
		}

		subSessionIndex := row - 1
		if subSessionIndex >= 0 && subSessionIndex < len(selectedSession.SubSessions) {
			selectedSubSession := selectedSession.SubSessions[subSessionIndex]

			// Build interruption details text
			var detailsText string
			if len(selectedSubSession.Interruptions) == 0 {
				detailsText = "No interruptions recorded for this sub-session."
			} else {
				detailsText = fmt.Sprintf("[yellow]Interruptions for Sub-Session #%d:[white]\n\n", subSessionIndex+1)

				for i := 0; i < len(selectedSubSession.Interruptions); i += 2 {
					interrupt := selectedSubSession.Interruptions[i]

					// Format interruption start
					interruptStart := fmt.Sprintf("[yellow]Start:[white] %s", models.FormatTime(interrupt.StartTime))

					// Format interruption type
					interruptType := string(interrupt.Tag)
					if interruptType == "" {
						interruptType = "Unknown"
					}
					interruptTypeStr := fmt.Sprintf("[yellow]Type:[white] %s", interruptType)

					// Format interruption description
					description := interrupt.Description
					if description == "" {
						description = "(No description)"
					}
					descriptionStr := fmt.Sprintf("[yellow]Description:[white] %s", description)

					// Format end time and duration if available
					durationStr := ""
					if i+1 < len(selectedSubSession.Interruptions) {
						returnEntry := selectedSubSession.Interruptions[i+1]
						interruptEnd := fmt.Sprintf("[yellow]End:[white] %s", models.FormatTime(returnEntry.StartTime))

						duration := returnEntry.StartTime.Sub(interrupt.StartTime)
						durationFormatted := formatDurationHumanReadable(duration)
						durationStr = fmt.Sprintf("[yellow]Duration:[white] %s", durationFormatted)

						detailsText += "Interruption #" + fmt.Sprint((i/2)+1) + ":\n" +
							interruptTypeStr + "\n" +
							descriptionStr + "\n" +
							interruptStart + "\n" +
							interruptEnd + "\n" +
							durationStr + "\n\n"
					} else {
						// Active interruption
						interruptEnd := fmt.Sprintf("[yellow]End:[white] [red]Active[white]")

						duration := time.Since(interrupt.StartTime)
						durationFormatted := formatDurationHumanReadable(duration)
						durationStr = fmt.Sprintf("[yellow]Duration:[white] %s (ongoing)", durationFormatted)

						detailsText += "Interruption #" + fmt.Sprint((i/2)+1) + ":\n" +
							interruptTypeStr + "\n" +
							descriptionStr + "\n" +
							interruptStart + "\n" +
							interruptEnd + "\n" +
							durationStr + "\n\n"
					}
				}
			}

			interruptionsText.SetText(detailsText)
		}
	})

	// Create a flex to ensure the modal has good dimensions
	modalWrapper := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(modalFlex, 70, 1, true).
			AddItem(nil, 0, 1, false),
			20, 1, true).
		AddItem(nil, 0, 1, false)

	// Set border and title
	modalFlex.SetBorder(true).
		SetTitle(" Session Details ").
		SetTitleAlign(tview.AlignCenter)

	// Add key capture for escape key and q/Q keys
	modalWrapper.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Rune() == 'q' || event.Rune() == 'Q' {
			ui.pages.RemovePage("session_details")
			ui.app.SetFocus(ui.sessionsTable)
			return nil
		}
		return event
	})

	// Show the modal
	ui.pages.AddPage("session_details", modalWrapper, true, true)
	ui.app.SetFocus(subSessionsTable)

	// Trigger the selection of the first sub-session to show its interruptions
	if len(selectedSession.SubSessions) > 0 {
		subSessionsTable.Select(1, 0)
	}
}

// calculateTableColumnWidths automatically calculates appropriate column widths
// for a table based on header text and content
func calculateTableColumnWidths(table *tview.Table) []int {
	if table.GetRowCount() == 0 {
		return nil
	}

	// Get number of columns
	columnCount := table.GetColumnCount()
	if columnCount == 0 {
		return nil
	}

	// Initialize column widths with minimum values
	columnWidths := make([]int, columnCount)
	for i := range columnWidths {
		columnWidths[i] = 10 // Minimum width (to accommodate padding)
	}

	// Determine maximum width needed for each column
	for row := 0; row < table.GetRowCount(); row++ {
		for col := 0; col < columnCount; col++ {
			cell := table.GetCell(row, col)
			if cell == nil {
				continue
			}

			text := cell.Text
			textWidth := len(text)

			// Update max width if this cell's content is wider
			if textWidth > columnWidths[col] {
				columnWidths[col] = textWidth
			}
		}
	}

	// Apply the widths to all cells in each column
	for col := 0; col < columnCount; col++ {
		width := columnWidths[col]
		for row := 0; row < table.GetRowCount(); row++ {
			cell := table.GetCell(row, col)
			if cell != nil {
				cell.SetMaxWidth(width)
			}
		}
	}

	return columnWidths
}
