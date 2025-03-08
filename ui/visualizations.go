package ui

import (
	"fmt"
	"sort"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/lukaszraczylo/interruption-tracker/models"
	"github.com/rivo/tview"
)

// Chart types
const (
	ChartTypeBar     = "bar"
	ChartTypeLine    = "line"
	ChartTypeHeatmap = "heatmap"
)

// VisualizationData contains data for rendering different types of charts
type VisualizationData struct {
	Title       string
	Description string
	ChartType   string
	Labels      []string
	Values      []float64
	ColorFunc   func(value float64) string // Function to determine color based on value
}

// renderBarChart creates a bar chart visualization
func renderBarChart(app *tview.Application, data *VisualizationData) *tview.Flex {
	// Create the chart content
	content := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)

	// Create header and description
	header := tview.NewTextView().
		SetTextColor(tcell.ColorGreen).
		SetText(fmt.Sprintf(" %s ", data.Title)).
		SetTextAlign(tview.AlignCenter)

	description := tview.NewTextView().
		SetTextColor(tcell.ColorWhite).
		SetText(fmt.Sprintf(" %s ", data.Description)).
		SetTextAlign(tview.AlignCenter)

	// Prepare data for chart
	if len(data.Labels) != len(data.Values) {
		content.SetText("Error: Data labels and values must have the same length")
		return tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(header, 1, 0, false).
			AddItem(description, 1, 0, false).
			AddItem(content, 0, 1, false)
	}

	// Find the maximum value for scaling
	var maxValue float64
	for _, value := range data.Values {
		if value > maxValue {
			maxValue = value
		}
	}

	// Create the chart text
	chartText := ""
	for i, label := range data.Labels {
		value := data.Values[i]

		// Determine bar size (max 40 characters)
		barWidth := int((value / maxValue) * 40)
		if barWidth < 1 && value > 0 {
			barWidth = 1 // Always show at least one character for non-zero values
		}

		// Create the bar
		bar := ""
		for j := 0; j < barWidth; j++ {
			bar += "█"
		}

		// Apply color if available
		barColor := "[blue]"
		if data.ColorFunc != nil {
			barColor = data.ColorFunc(value)
		}

		// Format the line with value and label
		chartText += fmt.Sprintf("[yellow]%-15s[white] %6.1f %s%s[white]\n", label, value, barColor, bar)
	}

	content.SetText(chartText)

	// Create flex layout
	chart := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(description, 1, 0, false).
		AddItem(content, 0, 1, false)

	return chart
}

// renderHeatmap creates a productivity heatmap visualization
// createInterruptionsChart creates a bar chart showing interruption counts by type
func createInterruptionsChart(app *tview.Application, stats *models.DetailedStats) *tview.Flex {
	// Convert interruptions by tag to sorted chart data
	var labels []string
	var values []float64

	for tag, count := range stats.InterruptionsByTag {
		labels = append(labels, string(tag))
		values = append(values, float64(count))
	}

	// Create VisualizationData
	data := &VisualizationData{
		Title:       "Interruptions by Type",
		Description: "Number of interruptions by category",
		ChartType:   ChartTypeBar,
		Labels:      labels,
		Values:      values,
		ColorFunc: func(value float64) string {
			// Lower values are better for interruption counts
			return createColorGradient(value, values[0], values[len(values)-1])
		},
	}

	return renderBarChart(app, data)
}

// createProductivityChart creates a bar chart showing productivity by hour of day
func createProductivityChart(app *tview.Application, stats *models.DetailedStats) *tview.Flex {
	// Convert hourly productivity to sorted chart data
	type hourData struct {
		hour  int
		value float64
	}

	hourlyValues := []hourData{}
	for hour, duration := range stats.HourlyProductivity {
		hourlyValues = append(hourlyValues, hourData{
			hour:  hour,
			value: float64(duration) / float64(time.Hour), // Convert to hours
		})
	}

	// Sort by hour
	sort.Slice(hourlyValues, func(i, j int) bool {
		return hourlyValues[i].hour < hourlyValues[j].hour
	})

	// Create chart data
	var labels []string
	var values []float64

	for _, data := range hourlyValues {
		hourStr := fmt.Sprintf("%d:00", data.hour)
		labels = append(labels, hourStr)
		values = append(values, data.value)
	}

	// Create VisualizationData
	data := &VisualizationData{
		Title:       "Productivity by Hour",
		Description: "Hours of focused work by time of day",
		ChartType:   ChartTypeBar,
		Labels:      labels,
		Values:      values,
		ColorFunc: func(value float64) string {
			// Higher values are better for productivity
			if len(values) <= 1 {
				return "[green]"
			}
			// Find min and max
			var min, max float64 = values[0], values[0]
			for _, v := range values {
				if v < min {
					min = v
				}
				if v > max {
					max = v
				}
			}
			return createColorGradient(value, min, max)
		},
	}

	return renderBarChart(app, data)
}

// createProductivityScoreView creates a view showing the calculated productivity score
func createProductivityScoreView(app *tview.Application, stats *models.DetailedStats) *tview.Flex {
	// Calculate score if not already done
	if stats.ProductivityScore == 0 {
		stats.CalculateProductivityScore()
	}

	// Create view
	scoreText := fmt.Sprintf("%.1f", stats.ProductivityScore)

	// Create score display
	scoreView := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	// Apply color based on score
	coloredScore := applyColorToText(scoreText, stats.ProductivityScore, 0, 100)

	// Create trend indicator
	trend := stats.GetProductivityTrend()
	trendIndicator := ""
	if trend > 0.1 {
		trendIndicator = " [green]↑ Improving"
	} else if trend < -0.1 {
		trendIndicator = " [red]↓ Declining"
	} else {
		trendIndicator = " [yellow]→ Stable"
	}

	// Create full score text
	fullScoreText := fmt.Sprintf("\n\n[white]Productivity Score (0-100):\n\n[::b]%s[::] %s\n\n", coloredScore, trendIndicator)

	// Add explanation of score
	explanation := "Score based on:\n" +
		"• Focused work time\n" +
		"• Interruption frequency\n" +
		"• Recovery time impact\n\n"

	// Add recommendations based on score
	recommendations := "[yellow]Recommendations:[white]\n"
	if stats.ProductivityScore < 40 {
		recommendations += "• Reduce interruptions\n• Consider time blocking\n• Create a do-not-disturb system"
	} else if stats.ProductivityScore < 70 {
		recommendations += "• Group similar tasks\n• Schedule focused work periods\n• Manage interruption sources"
	} else {
		recommendations += "• Maintain current work patterns\n• Consider optimizing work hours\n• Share techniques with team"
	}

	scoreView.SetText(fullScoreText + explanation + recommendations)

	// Create header
	header := tview.NewTextView().
		SetTextColor(tcell.ColorGreen).
		SetText(" Productivity Analysis ").
		SetTextAlign(tview.AlignCenter)

	// Create flex layout
	scoreContainer := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(scoreView, 0, 1, false)

	return scoreContainer
}

// createDailyProductivityChart creates a chart showing daily productivity
func createDailyProductivityChart(app *tview.Application, stats *models.DetailedStats) *tview.Flex {
	// Convert daily work durations to chart data
	type dayData struct {
		date  string
		value float64
	}

	var dailyValues []dayData
	for dateStr, duration := range stats.DailyWorkDurations {
		dailyValues = append(dailyValues, dayData{
			date:  dateStr,
			value: float64(duration) / float64(time.Hour), // Convert to hours
		})
	}

	// Sort by date
	sort.Slice(dailyValues, func(i, j int) bool {
		return dailyValues[i].date < dailyValues[j].date
	})

	// Take only the last 10 days if we have more
	if len(dailyValues) > 10 {
		dailyValues = dailyValues[len(dailyValues)-10:]
	}

	// Create chart data
	var labels []string
	var values []float64

	for _, data := range dailyValues {
		// Format date as day-month only
		t, err := time.Parse("2006-01-02", data.date)
		if err == nil {
			labels = append(labels, t.Format("02-Jan"))
		} else {
			labels = append(labels, data.date)
		}
		values = append(values, data.value)
	}

	// Create VisualizationData
	data := &VisualizationData{
		Title:       "Daily Productivity",
		Description: "Hours of focused work by day",
		ChartType:   ChartTypeBar,
		Labels:      labels,
		Values:      values,
		ColorFunc: func(value float64) string {
			// Higher values are better for productivity
			if len(values) <= 1 {
				return "[green]"
			}
			// Find min and max
			var min, max float64 = values[0], values[0]
			for _, v := range values {
				if v < min {
					min = v
				}
				if v > max {
					max = v
				}
			}
			return createColorGradient(value, min, max)
		},
	}

	return renderBarChart(app, data)
}
