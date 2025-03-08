package models

import (
	"time"
)

// DetailedStats contains comprehensive statistics for analysis
type DetailedStats struct {
	// Date range for the statistics
	StartDate time.Time
	EndDate   time.Time

	// Overall work stats
	TotalWorkDuration  time.Duration
	TotalSessions      int
	LongestSession     time.Duration
	AverageSessionTime time.Duration

	// Interruption stats
	TotalInterruptions        int
	InterruptionsByTag        map[InterruptionTag]int
	InterruptionDurationByTag map[InterruptionTag]time.Duration

	// Time analysis
	DailyWorkDurations map[string]time.Duration // Map of date string to duration
	HourlyProductivity map[int]time.Duration    // Map of hour (0-23) to duration

	// Generated metrics
	ProductivityScore float64 // 0-100 score based on focus time vs interruptions
}

// CalculateProductivityScore computes a productivity score based on work and interruption patterns
func (s *DetailedStats) CalculateProductivityScore() float64 {
	if s.TotalWorkDuration == 0 {
		return 0
	}

	// Calculate total interruption time
	var totalInterruptionTime time.Duration
	for _, duration := range s.InterruptionDurationByTag {
		totalInterruptionTime += duration
	}

	// Calculate recovery time (10 minutes per interruption)
	recoveryTime := time.Duration(s.TotalInterruptions) * 10 * time.Minute

	// Total impacted time
	totalImpactedTime := totalInterruptionTime + recoveryTime

	// Calculate work ratio (pure work time / total time)
	totalTime := s.TotalWorkDuration + totalImpactedTime
	workRatio := float64(s.TotalWorkDuration) / float64(totalTime)

	// Convert to 0-100 score
	score := workRatio * 100

	// Apply penalties for too many interruptions
	interruptionRatio := float64(s.TotalInterruptions) / float64(s.TotalSessions)
	if interruptionRatio > 0.5 {
		// Apply penalty for high interruption rate
		penaltyFactor := (interruptionRatio - 0.5) * 0.2 // Up to 20% penalty
		score = score * (1 - penaltyFactor)
	}

	// Cap the score at 100
	if score > 100 {
		score = 100
	}

	s.ProductivityScore = score
	return score
}

// GetMostProductiveHour returns the hour with the highest productivity
func (s *DetailedStats) GetMostProductiveHour() (hour int, duration time.Duration) {
	var maxDuration time.Duration
	var maxHour int

	for h, d := range s.HourlyProductivity {
		if d > maxDuration {
			maxDuration = d
			maxHour = h
		}
	}

	return maxHour, maxDuration
}

// GetInterruptionBreakdown returns a breakdown of interruptions by type
func (s *DetailedStats) GetInterruptionBreakdown() []InterruptionTagStats {
	result := make([]InterruptionTagStats, 0, len(s.InterruptionsByTag))

	for tag, count := range s.InterruptionsByTag {
		duration := s.InterruptionDurationByTag[tag]
		recoveryTime := time.Duration(count) * 10 * time.Minute

		stats := InterruptionTagStats{
			Tag:               tag,
			Count:             count,
			TotalTime:         duration,
			RecoveryTime:      recoveryTime,
			TotalWithRecovery: duration + recoveryTime,
		}

		if count > 0 {
			stats.AverageTime = duration / time.Duration(count)
		}

		result = append(result, stats)
	}

	return result
}

// GetProductivityTrend calculates the trend in productivity over the date range
func (s *DetailedStats) GetProductivityTrend() float64 {
	if len(s.DailyWorkDurations) <= 1 {
		return 0 // Not enough data for trend
	}

	type dayData struct {
		date  time.Time
		value float64
	}

	// Convert map to slice for sorting
	days := make([]dayData, 0, len(s.DailyWorkDurations))
	for dateStr, duration := range s.DailyWorkDurations {
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		// Convert to hours
		hours := float64(duration) / float64(time.Hour)
		days = append(days, dayData{date, hours})
	}

	// Need at least 2 days for trend
	if len(days) < 2 {
		return 0
	}

	// Simple linear regression
	var sumX, sumY, sumXY, sumX2 float64
	n := float64(len(days))

	// Use days since start as X
	startDate := s.StartDate
	for _, day := range days {
		x := float64(day.date.Sub(startDate).Hours() / 24) // Days since start
		y := day.value                                     // Hours worked

		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// Calculate slope
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	return slope // Positive = improving, negative = declining
}
