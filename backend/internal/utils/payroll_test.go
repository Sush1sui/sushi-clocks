package utils

import (
	"math"
	"testing"
	"time"
)

func TestComputeShiftDurationMinutes(t *testing.T) {
	t.Run("same day duration", func(t *testing.T) {
		in := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
		out := time.Date(2026, 10, 1, 17, 0, 0, 0, time.UTC)
		mins, err := ComputeShiftDurationMinutes(in, out, "UTC")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mins != 480 {
			t.Errorf("expected 480, got %f", mins)
		}
	})

	t.Run("cross midnight duration", func(t *testing.T) {
		in := time.Date(2026, 10, 1, 22, 0, 0, 0, time.UTC)
		out := time.Date(2026, 10, 2, 6, 0, 0, 0, time.UTC)
		mins, err := ComputeShiftDurationMinutes(in, out, "Asia/Manila")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mins != 480 {
			t.Errorf("expected 480, got %f", mins)
		}
	})

	t.Run("clock_out before clock_in returns error", func(t *testing.T) {
		in := time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)
		out := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
		_, err := ComputeShiftDurationMinutes(in, out, "UTC")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("invalid timezone returns error", func(t *testing.T) {
		in := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
		out := time.Date(2026, 10, 1, 17, 0, 0, 0, time.UTC)
		_, err := ComputeShiftDurationMinutes(in, out, "Invalid/Timezone_XYZ")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestSplitShiftByCalendarDay(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Manila")
	if err != nil {
		t.Fatalf("failed to load Asia/Manila location: %v", err)
	}

	t.Run("single day shift in local timezone", func(t *testing.T) {
		in := time.Date(2026, 10, 1, 9, 0, 0, 0, loc)
		out := time.Date(2026, 10, 1, 17, 0, 0, 0, loc)
		segments, err := SplitShiftByCalendarDay(in, out, "Asia/Manila")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(segments) != 1 {
			t.Fatalf("expected 1 segment, got %d", len(segments))
		}
		if segments[0].MinutesWorked != 480 {
			t.Errorf("expected 480 mins, got %f", segments[0].MinutesWorked)
		}
		expectedDate := time.Date(2026, 10, 1, 0, 0, 0, 0, loc)
		if !segments[0].LocalDate.Equal(expectedDate) {
			t.Errorf("expected date %v, got %v", expectedDate, segments[0].LocalDate)
		}
	})

	t.Run("overnight shift cross-midnight in local timezone", func(t *testing.T) {
		// 10:00 PM Oct 1 to 6:00 AM Oct 2
		in := time.Date(2026, 10, 1, 22, 0, 0, 0, loc)
		out := time.Date(2026, 10, 2, 6, 0, 0, 0, loc)
		segments, err := SplitShiftByCalendarDay(in, out, "Asia/Manila")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(segments) != 2 {
			t.Fatalf("expected 2 segments, got %d", len(segments))
		}

		// Segment 1: Oct 1, 22:00 to 24:00 (120 mins)
		expectedDate1 := time.Date(2026, 10, 1, 0, 0, 0, 0, loc)
		if !segments[0].LocalDate.Equal(expectedDate1) {
			t.Errorf("expected seg 0 date %v, got %v", expectedDate1, segments[0].LocalDate)
		}
		if segments[0].MinutesWorked != 120 {
			t.Errorf("expected seg 0 120 mins, got %f", segments[0].MinutesWorked)
		}

		// Segment 2: Oct 2, 00:00 to 06:00 (360 mins)
		expectedDate2 := time.Date(2026, 10, 2, 0, 0, 0, 0, loc)
		if !segments[1].LocalDate.Equal(expectedDate2) {
			t.Errorf("expected seg 1 date %v, got %v", expectedDate2, segments[1].LocalDate)
		}
		if segments[1].MinutesWorked != 360 {
			t.Errorf("expected seg 1 360 mins, got %f", segments[1].MinutesWorked)
		}
	})

	t.Run("spans 3 calendar days", func(t *testing.T) {
		in := time.Date(2026, 10, 1, 23, 0, 0, 0, loc)
		out := time.Date(2026, 10, 3, 2, 0, 0, 0, loc)
		segments, err := SplitShiftByCalendarDay(in, out, "Asia/Manila")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(segments) != 3 {
			t.Fatalf("expected 3 segments, got %d", len(segments))
		}
		// Day 1: 60 mins
		if segments[0].MinutesWorked != 60 {
			t.Errorf("expected seg 0 60 mins, got %f", segments[0].MinutesWorked)
		}
		// Day 2: 1440 mins (full day)
		if segments[1].MinutesWorked != 1440 {
			t.Errorf("expected seg 1 1440 mins, got %f", segments[1].MinutesWorked)
		}
		// Day 3: 120 mins
		if segments[2].MinutesWorked != 120 {
			t.Errorf("expected seg 2 120 mins, got %f", segments[2].MinutesWorked)
		}
	})

	t.Run("clock_out equals clock_in returns empty slice", func(t *testing.T) {
		in := time.Date(2026, 10, 1, 9, 0, 0, 0, loc)
		segments, err := SplitShiftByCalendarDay(in, in, "Asia/Manila")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(segments) != 0 {
			t.Errorf("expected 0 segments, got %d", len(segments))
		}
	})

	t.Run("clock_out before clock_in returns error", func(t *testing.T) {
		in := time.Date(2026, 10, 2, 9, 0, 0, 0, loc)
		out := time.Date(2026, 10, 1, 9, 0, 0, 0, loc)
		_, err := SplitShiftByCalendarDay(in, out, "Asia/Manila")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestCountWorkingDaysInPeriod(t *testing.T) {
	// 2026-10-05 is Monday, 2026-10-18 is Sunday (exactly 14 calendar days, 2 full weeks)
	start := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		mask     int
		expected int
	}{
		{"Mon-Fri (mask 62)", 62, 10},
		{"Mon-Sat (mask 126)", 126, 12},
		{"All 7 days (mask 127)", 127, 14},
		{"Weekends only (mask 65)", 65, 4},
		{"Default when mask <= 0 (Mon-Fri)", 0, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountWorkingDaysInPeriod(start, end, tt.mask)
			if got != tt.expected {
				t.Errorf("CountWorkingDaysInPeriod() = %d, want %d", got, tt.expected)
			}
		})
	}

	t.Run("end before start returns 0", func(t *testing.T) {
		got := CountWorkingDaysInPeriod(end, start, 62)
		if got != 0 {
			t.Errorf("expected 0, got %d", got)
		}
	})
}

func TestClipLeaveDaysToPeriod(t *testing.T) {
	periodStart := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		leaveStart time.Time
		leaveEnd   time.Time
		expected   int
	}{
		{
			name:       "fully within period",
			leaveStart: time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC),
			leaveEnd:   time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC),
			expected:   4, // Oct 5, 6, 7, 8
		},
		{
			name:       "partial left overlap",
			leaveStart: time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC),
			leaveEnd:   time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC),
			expected:   3, // Oct 1, 2, 3
		},
		{
			name:       "partial right overlap",
			leaveStart: time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC),
			leaveEnd:   time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC),
			expected:   4, // Oct 12, 13, 14, 15
		},
		{
			name:       "spans entire period and beyond",
			leaveStart: time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC),
			leaveEnd:   time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC),
			expected:   15, // Oct 1 through Oct 15
		},
		{
			name:       "strictly before period",
			leaveStart: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
			leaveEnd:   time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
			expected:   0,
		},
		{
			name:       "strictly after period",
			leaveStart: time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC),
			leaveEnd:   time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC),
			expected:   0,
		},
		{
			name:       "single day on period boundary",
			leaveStart: time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC),
			leaveEnd:   time.Date(2026, 10, 15, 23, 59, 59, 0, time.UTC),
			expected:   1,
		},
		{
			name:       "inverted leave dates",
			leaveStart: time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC),
			leaveEnd:   time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC),
			expected:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClipLeaveDaysToPeriod(tt.leaveStart, tt.leaveEnd, periodStart, periodEnd)
			if got != tt.expected {
				t.Errorf("ClipLeaveDaysToPeriod() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestProrateMonthlyWage(t *testing.T) {
	// Full month of January 2026 (31 calendar days, 22 working days Mon-Fri)
	janStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	janEnd := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)

	t.Run("calendar_days basis", func(t *testing.T) {
		wage, err := ProrateMonthlyWage(31000, ProrateCalendarDays, janStart, janEnd, 15, 62)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// (31000 / 31) * 15 = 15000.00
		if wage != 15000.00 {
			t.Errorf("expected 15000.00, got %f", wage)
		}
	})

	t.Run("working_days basis", func(t *testing.T) {
		// Jan 2026 has 22 working days (Mon-Fri)
		wage, err := ProrateMonthlyWage(22000, ProrateWorkingDays, janStart, janEnd, 11, 62)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// (22000 / 22) * 11 = 11000.00
		if wage != 11000.00 {
			t.Errorf("expected 11000.00, got %f", wage)
		}
	})

	t.Run("fixed_22 basis", func(t *testing.T) {
		wage, err := ProrateMonthlyWage(22000, ProrateFixed22, janStart, janEnd, 11, 62)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// (22000 / 22) * 11 = 11000.00
		if wage != 11000.00 {
			t.Errorf("expected 11000.00, got %f", wage)
		}
	})

	t.Run("fixed_26 basis", func(t *testing.T) {
		wage, err := ProrateMonthlyWage(26000, ProrateFixed26, janStart, janEnd, 13, 62)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// (26000 / 26) * 13 = 13000.00
		if wage != 13000.00 {
			t.Errorf("expected 13000.00, got %f", wage)
		}
	})

	t.Run("rounding precision to 2 decimal places", func(t *testing.T) {
		// (10000 / 22) * 7 = 3181.81818... -> 3181.82
		wage, err := ProrateMonthlyWage(10000, ProrateFixed22, janStart, janEnd, 7, 62)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if math.Abs(wage-3181.82) > 0.001 {
			t.Errorf("expected 3181.82, got %f", wage)
		}
	})

	t.Run("negative worked days returns error", func(t *testing.T) {
		_, err := ProrateMonthlyWage(10000, ProrateFixed22, janStart, janEnd, -1, 62)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("negative monthly wage returns error", func(t *testing.T) {
		_, err := ProrateMonthlyWage(-10000, ProrateFixed22, janStart, janEnd, 5, 62)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("unsupported basis returns error", func(t *testing.T) {
		_, err := ProrateMonthlyWage(10000, "unknown_basis", janStart, janEnd, 5, 62)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
