package utils

import (
	"errors"
	"fmt"
	"math"
	"time"
)

const (
	ProrateCalendarDays = "calendar_days"
	ProrateWorkingDays  = "working_days"
	ProrateFixed22      = "fixed_22"
	ProrateFixed26      = "fixed_26"
)

type DaySegment struct {
	LocalDate     time.Time
	MinutesWorked float64
}

// ComputeShiftDurationMinutes returns the total duration between clockIn and clockOut in minutes.
// Validates timezone if provided.
func ComputeShiftDurationMinutes(clockIn, clockOut time.Time, timezone string) (float64, error) {
	if timezone != "" {
		if _, err := time.LoadLocation(timezone); err != nil {
			return 0, fmt.Errorf("invalid timezone: %w", err)
		}
	}
	if clockOut.Before(clockIn) {
		return 0, errors.New("clock_out cannot be before clock_in")
	}
	return clockOut.Sub(clockIn).Minutes(), nil
}

// SplitShiftByCalendarDay splits a shift spanning multiple calendar days into daily segments
// based on midnight boundaries in the company's local timezone.
func SplitShiftByCalendarDay(clockIn, clockOut time.Time, timezone string) ([]DaySegment, error) {
	loc := time.UTC
	if timezone != "" {
		var err error
		loc, err = time.LoadLocation(timezone)
		if err != nil {
			return nil, fmt.Errorf("invalid timezone: %w", err)
		}
	}

	if clockOut.Before(clockIn) {
		return nil, errors.New("clock_out cannot be before clock_in")
	}
	if clockOut.Equal(clockIn) {
		return []DaySegment{}, nil
	}

	inLocal := clockIn.In(loc)
	outLocal := clockOut.In(loc)

	var segments []DaySegment
	curr := inLocal

	for curr.Before(outLocal) {
		nextMidnight := time.Date(curr.Year(), curr.Month(), curr.Day()+1, 0, 0, 0, 0, loc)
		segEnd := nextMidnight
		if outLocal.Before(nextMidnight) {
			segEnd = outLocal
		}

		mins := segEnd.Sub(curr).Minutes()
		dayDate := time.Date(curr.Year(), curr.Month(), curr.Day(), 0, 0, 0, 0, loc)
		segments = append(segments, DaySegment{
			LocalDate:     dayDate,
			MinutesWorked: mins,
		})

		curr = segEnd
	}

	return segments, nil
}

// CountWorkingDaysInPeriod counts how many days between periodStart and periodEnd (inclusive)
// match the workDaysMask. Bit position corresponds to time.Weekday (0=Sun, 1=Mon, ..., 6=Sat).
func CountWorkingDaysInPeriod(periodStart, periodEnd time.Time, workDaysMask int) int {
	if workDaysMask <= 0 {
		workDaysMask = 62 // Default to Mon-Fri
	}

	start := time.Date(periodStart.Year(), periodStart.Month(), periodStart.Day(), 0, 0, 0, 0, periodStart.Location())
	end := time.Date(periodEnd.Year(), periodEnd.Month(), periodEnd.Day(), 0, 0, 0, 0, periodEnd.Location())
	if end.Before(start) {
		return 0
	}

	count := 0
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		weekday := int(d.Weekday())
		if (workDaysMask & (1 << weekday)) != 0 {
			count++
		}
	}
	return count
}

// ClipLeaveDaysToPeriod returns the number of inclusive leave days that fall within the pay period.
// If there is no overlap, returns 0.
func ClipLeaveDaysToPeriod(leaveStart, leaveEnd, periodStart, periodEnd time.Time) int {
	lStart := time.Date(leaveStart.Year(), leaveStart.Month(), leaveStart.Day(), 0, 0, 0, 0, time.UTC)
	lEnd := time.Date(leaveEnd.Year(), leaveEnd.Month(), leaveEnd.Day(), 0, 0, 0, 0, time.UTC)
	pStart := time.Date(periodStart.Year(), periodStart.Month(), periodStart.Day(), 0, 0, 0, 0, time.UTC)
	pEnd := time.Date(periodEnd.Year(), periodEnd.Month(), periodEnd.Day(), 0, 0, 0, 0, time.UTC)

	if lEnd.Before(lStart) || pEnd.Before(pStart) {
		return 0
	}

	overlapStart := lStart
	if pStart.After(overlapStart) {
		overlapStart = pStart
	}

	overlapEnd := lEnd
	if pEnd.Before(overlapEnd) {
		overlapEnd = pEnd
	}

	if overlapStart.After(overlapEnd) {
		return 0
	}

	return int(overlapEnd.Sub(overlapStart).Hours()/24) + 1
}

// ProrateMonthlyWage calculates the prorated portion of a monthly salary based on the chosen basis.
func ProrateMonthlyWage(monthlyWage float64, basis string, periodStart, periodEnd time.Time, workedDays int, workDaysMask int) (float64, error) {
	if workedDays < 0 {
		return 0, errors.New("worked_days cannot be negative")
	}
	if monthlyWage < 0 {
		return 0, errors.New("monthly_wage cannot be negative")
	}
	if basis == "" {
		basis = ProrateCalendarDays
	}

	var totalDays float64

	switch basis {
	case ProrateCalendarDays:
		pStart := time.Date(periodStart.Year(), periodStart.Month(), periodStart.Day(), 0, 0, 0, 0, time.UTC)
		pEnd := time.Date(periodEnd.Year(), periodEnd.Month(), periodEnd.Day(), 0, 0, 0, 0, time.UTC)
		if pEnd.Before(pStart) {
			return 0, errors.New("period_end cannot be before period_start")
		}
		totalDays = float64(int(pEnd.Sub(pStart).Hours()/24) + 1)

	case ProrateWorkingDays:
		pStart := time.Date(periodStart.Year(), periodStart.Month(), periodStart.Day(), 0, 0, 0, 0, time.UTC)
		pEnd := time.Date(periodEnd.Year(), periodEnd.Month(), periodEnd.Day(), 0, 0, 0, 0, time.UTC)
		if pEnd.Before(pStart) {
			return 0, errors.New("period_end cannot be before period_start")
		}
		totalDays = float64(CountWorkingDaysInPeriod(periodStart, periodEnd, workDaysMask))

	case ProrateFixed22:
		totalDays = 22.0

	case ProrateFixed26:
		totalDays = 26.0

	default:
		return 0, fmt.Errorf("unsupported proration basis: %s", basis)
	}

	if totalDays <= 0 {
		return 0, errors.New("proration denominator is zero")
	}

	prorated := (monthlyWage / totalDays) * float64(workedDays)
	return math.Round(prorated*100) / 100, nil
}
