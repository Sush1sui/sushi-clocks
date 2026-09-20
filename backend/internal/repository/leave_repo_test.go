package repository

import (
	"testing"
	"time"
)

func TestCalculateLeavePeriod(t *testing.T) {
	tests := []struct {
		name       string
		resetMonth int
		resetDay   int
		now        time.Time
		wantStart  time.Time
		wantEnd    time.Time
	}{
		{
			name:       "Jan 1 calendar reset - currently in September",
			resetMonth: 1,
			resetDay:   1,
			now:        time.Date(2026, 9, 13, 10, 0, 0, 0, time.UTC),
			wantStart:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:    time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name:       "July 1 fiscal reset - currently in September (after reset)",
			resetMonth: 7,
			resetDay:   1,
			now:        time.Date(2026, 9, 13, 10, 0, 0, 0, time.UTC),
			wantStart:  time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:    time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC),
		},
		{
			name:       "July 1 fiscal reset - currently in March (before reset)",
			resetMonth: 7,
			resetDay:   1,
			now:        time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC),
			wantStart:  time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:    time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			period := CalculateLeavePeriod(tt.resetMonth, tt.resetDay, tt.now)
			if !period.StartDate.Equal(tt.wantStart) {
				t.Errorf("got StartDate %v, want %v", period.StartDate, tt.wantStart)
			}
			if !period.EndDate.Equal(tt.wantEnd) {
				t.Errorf("got EndDate %v, want %v", period.EndDate, tt.wantEnd)
			}
		})
	}
}
