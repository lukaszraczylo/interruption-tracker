package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// RangeType represents the time range for visualization data
type RangeType string

const (
	RangeDay   RangeType = "day"
	RangeWeek  RangeType = "week"
	RangeMonth RangeType = "month"
)

// createVisualizationPages creates all visualization pages for the UI
func (ui *TimerUI) createVisualizationPages() {
	// Default to day view
	ui.createVisualizationPagesWithRange(RangeDay)
}

// createVisualizationPagesWithRange creates all visualization pages for a specific time range
func (ui *TimerUI) createVisualizationPagesWithRange(rangeType RangeType) {
	// Get detailed stats for visualizations
	detailedStats, err := ui.storage.GetDetailedStats(string(rangeType))
	if err != nil {
		// Just return if there's an error - we'll handle this gracefully
		return
	}

	// Format range for display
	rangeDisplay := map[RangeType]string{
		RangeDay:   "Today",
		RangeWeek:  "This Week",
		RangeMonth: "This Month",
	}[rangeType]

	// Create productivity page with charts
	productivityPage := tview.NewFlex().SetDirection(tview.FlexRow)

	// Add title with range
	title := tview.NewTextView().
		SetTextColor(tcell.ColorGreen).
		SetText(fmt.Sprintf(" Productivity Visualizations (%s) ", rangeDisplay)).
		SetTextAlign(tview.AlignCenter)
	productivityPage.AddItem(title, 1, 0, false)

	// Add range selector
	rangeSelector := tview.NewTextView().
		SetText(" Press (d) for day, (w) for week, (m) for month ").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(tcell.ColorBlue)
	productivityPage.AddItem(rangeSelector, 1, 0, false)

	// Add navigation instructions
	nav := tview.NewTextView().
		SetText(" Press (b) to return to main stats, (q) to quit, arrow keys to navigate ").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(tcell.ColorYellow)

	// Create horizontal container for charts
	chartContainer := tview.NewFlex().SetDirection(tview.FlexColumn)

	// Create productivity score chart
	scoreView := createProductivityScoreView(ui.app, detailedStats)
	chartContainer.AddItem(scoreView, 0, 1, true)

	// Create productivity by hour chart
	hourChart := createProductivityChart(ui.app, detailedStats)
	chartContainer.AddItem(hourChart, 0, 1, false)

	// Add charts to the page
	productivityPage.AddItem(chartContainer, 0, 1, true)
	productivityPage.AddItem(nav, 1, 0, false)

	// Create interruptions page with charts
	interruptionsPage := tview.NewFlex().SetDirection(tview.FlexRow)

	// Add title with range
	interTitle := tview.NewTextView().
		SetTextColor(tcell.ColorGreen).
		SetText(fmt.Sprintf(" Interruption Analysis (%s) ", rangeDisplay)).
		SetTextAlign(tview.AlignCenter)
	interruptionsPage.AddItem(interTitle, 1, 0, false)

	// Add range selector
	interRangeSelector := tview.NewTextView().
		SetText(" Press (d) for day, (w) for week, (m) for month ").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(tcell.ColorBlue)
	interruptionsPage.AddItem(interRangeSelector, 1, 0, false)

	// Create interruptions chart
	interChart := createInterruptionsChart(ui.app, detailedStats)
	interruptionsPage.AddItem(interChart, 0, 1, true)

	// Add navigation help
	interNav := tview.NewTextView().
		SetText(" Press (b) to return to main stats, (q) to quit ").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(tcell.ColorYellow)
	interruptionsPage.AddItem(interNav, 1, 0, false)

	// Create time trends page with daily chart
	trendsPage := tview.NewFlex().SetDirection(tview.FlexRow)

	// Add title with range
	trendsTitle := tview.NewTextView().
		SetTextColor(tcell.ColorGreen).
		SetText(fmt.Sprintf(" Productivity Trends (%s) ", rangeDisplay)).
		SetTextAlign(tview.AlignCenter)
	trendsPage.AddItem(trendsTitle, 1, 0, false)

	// Add range selector
	trendsRangeSelector := tview.NewTextView().
		SetText(" Press (d) for day, (w) for week, (m) for month ").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(tcell.ColorBlue)
	trendsPage.AddItem(trendsRangeSelector, 1, 0, false)

	// Create daily chart if we have enough data
	if len(detailedStats.DailyWorkDurations) > 0 {
		dailyChart := createDailyProductivityChart(ui.app, detailedStats)
		trendsPage.AddItem(dailyChart, 0, 1, true)
	} else {
		// Show placeholder if not enough data
		noData := tview.NewTextView().
			SetText("Not enough historical data available to display trends.\nTrack more days to see productivity patterns over time.").
			SetTextAlign(tview.AlignCenter)
		trendsPage.AddItem(noData, 0, 1, true)
	}

	// Add navigation help
	trendsNav := tview.NewTextView().
		SetText(" Press (b) to return to main stats, (q) to quit ").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(tcell.ColorYellow)
	trendsPage.AddItem(trendsNav, 1, 0, false)

	// Add direct input capture to each visualization page to ensure q/Q works, 'b' to go back, and range selection works
	productivityPage.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' || event.Rune() == 'Q' {
			ui.app.Stop()
			return nil
		} else if event.Rune() == 'b' || event.Rune() == 'B' {
			ui.pages.SwitchToPage("stats")
			return nil
		} else if event.Rune() == 'd' || event.Rune() == 'D' {
			// Switch to day view
			ui.updateVisualizationPages(RangeDay)
			return nil
		} else if event.Rune() == 'w' || event.Rune() == 'W' {
			// Switch to week view
			ui.updateVisualizationPages(RangeWeek)
			return nil
		} else if event.Rune() == 'm' || event.Rune() == 'M' {
			// Switch to month view
			ui.updateVisualizationPages(RangeMonth)
			return nil
		}
		return event
	})

	interruptionsPage.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' || event.Rune() == 'Q' {
			ui.app.Stop()
			return nil
		} else if event.Rune() == 'b' || event.Rune() == 'B' {
			ui.pages.SwitchToPage("stats")
			return nil
		} else if event.Rune() == 'd' || event.Rune() == 'D' {
			// Switch to day view
			ui.updateVisualizationPages(RangeDay)
			return nil
		} else if event.Rune() == 'w' || event.Rune() == 'W' {
			// Switch to week view
			ui.updateVisualizationPages(RangeWeek)
			return nil
		} else if event.Rune() == 'm' || event.Rune() == 'M' {
			// Switch to month view
			ui.updateVisualizationPages(RangeMonth)
			return nil
		}
		return event
	})

	trendsPage.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' || event.Rune() == 'Q' {
			ui.app.Stop()
			return nil
		} else if event.Rune() == 'b' || event.Rune() == 'B' {
			ui.pages.SwitchToPage("stats")
			return nil
		} else if event.Rune() == 'd' || event.Rune() == 'D' {
			// Switch to day view
			ui.updateVisualizationPages(RangeDay)
			return nil
		} else if event.Rune() == 'w' || event.Rune() == 'W' {
			// Switch to week view
			ui.updateVisualizationPages(RangeWeek)
			return nil
		} else if event.Rune() == 'm' || event.Rune() == 'M' {
			// Switch to month view
			ui.updateVisualizationPages(RangeMonth)
			return nil
		}
		return event
	})

	// Add pages to the UI
	ui.pages.AddPage("productivity", productivityPage, true, false)
	ui.pages.AddPage("interruptions", interruptionsPage, true, false)
	ui.pages.AddPage("trends", trendsPage, true, false)
}

// extendedKeyHandler extends the Key Handler with visualization controls
func (ui *TimerUI) extendedKeyHandler(event *tcell.EventKey) bool {
	// Get current page
	currentPage, _ := ui.pages.GetFrontPage()

	// Handle keys for stats and visualization pages
	switch currentPage {
	case "stats":
		// Add viz navigation from stats page
		switch event.Rune() {
		case 'p', 'P':
			ui.pages.SwitchToPage("productivity")
			return true
		case 'i', 'I':
			if !ui.isInInterruptionMode() {
				ui.pages.SwitchToPage("interruptions")
				return true
			}
			// If we're in interruption mode, don't handle 'i'
			return false
		case 't', 'T':
			ui.pages.SwitchToPage("trends")
			return true
		case 'h', 'H': // Alternative for 'p'
			ui.pages.SwitchToPage("productivity")
			return true
		}
	case "productivity", "interruptions", "trends":
		// Navigate back from viz pages
		switch event.Rune() {
		case 'b', 'B':
			ui.pages.SwitchToPage("stats")
			return true
		case 'q', 'Q':
			ui.app.Stop()
			return true
		}

		// Handle left/right navigation between viz pages
		switch event.Key() {
		case tcell.KeyLeft:
			switch currentPage {
			case "productivity":
				ui.pages.SwitchToPage("trends")
			case "interruptions":
				ui.pages.SwitchToPage("productivity")
			case "trends":
				ui.pages.SwitchToPage("interruptions")
			}
			return true
		case tcell.KeyRight:
			switch currentPage {
			case "productivity":
				ui.pages.SwitchToPage("interruptions")
			case "interruptions":
				ui.pages.SwitchToPage("trends")
			case "trends":
				ui.pages.SwitchToPage("productivity")
			}
			return true
		}
	}

	// Not handled by extended key handler
	return false
}

// updateVisualizationPages updates all visualization pages with a new range
func (ui *TimerUI) updateVisualizationPages(rangeType RangeType) {
	// Get current page to restore it after update
	currentPage, _ := ui.pages.GetFrontPage()

	// Remove existing visualization pages
	ui.pages.RemovePage("productivity")
	ui.pages.RemovePage("interruptions")
	ui.pages.RemovePage("trends")

	// Recreate with new range
	ui.createVisualizationPagesWithRange(rangeType)

	// Restore the page that was active
	if currentPage == "productivity" || currentPage == "interruptions" || currentPage == "trends" {
		ui.pages.SwitchToPage(currentPage)
	}
}

// isInInterruptionMode checks if the user is currently recording an interruption
// to avoid confusion with the interruption visualization page
func (ui *TimerUI) isInInterruptionMode() bool {
	// Check if there's an active session with an odd number of interruptions
	if ui.activeSession != nil && len(ui.activeSession.Interruptions) > 0 {
		return len(ui.activeSession.Interruptions)%2 != 0
	}
	return false
}
