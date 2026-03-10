package milestone

import (
	"testing"
	"time"
)

func TestNextMonday(t *testing.T) {
	result := NextMonday()

	// Verify format is YYYY-MM-DD
	parsed, err := time.Parse("2006-01-02", result)
	if err != nil {
		t.Fatalf("NextMonday() returned invalid date format: %q, err: %v", result, err)
	}

	// Verify it's a Monday
	if parsed.Weekday() != time.Monday {
		t.Errorf("NextMonday() = %q, weekday = %v, want Monday", result, parsed.Weekday())
	}

	// Verify it's in the future
	today := time.Now().Truncate(24 * time.Hour)
	if !parsed.After(today) {
		t.Errorf("NextMonday() = %q, want a date after today %v", result, today.Format("2006-01-02"))
	}
}

func TestWeeksUntilEndOfYear(t *testing.T) {
	tests := []struct {
		name      string
		startDate string
		want      int
	}{
		{
			name:      "start at Jan 1",
			startDate: "2026-01-01",
			want:      52, // 364 days / 7 = 52.0
		},
		{
			name:      "start at Dec 25",
			startDate: "2026-12-25",
			want:      1,
		},
		{
			name:      "start at Jul 1 2026",
			startDate: "2026-07-01",
			want:      27,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WeeksUntilEndOfYear(tt.startDate)
			if got != tt.want {
				t.Errorf("WeeksUntilEndOfYear(%q) = %d, want %d", tt.startDate, got, tt.want)
			}
		})
	}
}

func TestISOWeek(t *testing.T) {
	tests := []struct {
		name string
		date time.Time
		want int
	}{
		{
			name: "2026-01-01 Thursday",
			date: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			want: 1,
		},
		{
			name: "2026-12-31 Thursday",
			date: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
			want: 53,
		},
		{
			name: "2025-12-29 Monday",
			date: time.Date(2025, 12, 29, 0, 0, 0, 0, time.UTC),
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ISOWeek(tt.date)
			if got != tt.want {
				t.Errorf("ISOWeek(%v) = %d, want %d", tt.date.Format("2006-01-02"), got, tt.want)
			}
		})
	}
}

func TestToUtcDueOn(t *testing.T) {
	tests := []struct {
		name     string
		dateStr  string
		timezone string
		want     string
		wantErr  bool
	}{
		{
			name:     "UTC timezone",
			dateStr:  "2026-01-04",
			timezone: "UTC",
			want:     "2026-01-04T23:59:59Z",
		},
		{
			name:     "Asia/Tokyo UTC+9",
			dateStr:  "2026-01-04",
			timezone: "Asia/Tokyo",
			want:     "2026-01-04T14:59:59Z",
		},
		{
			name:     "America/New_York UTC-5 winter",
			dateStr:  "2026-01-04",
			timezone: "America/New_York",
			want:     "2026-01-05T04:59:59Z",
		},
		{
			name:     "invalid timezone",
			dateStr:  "2026-01-04",
			timezone: "Invalid/Timezone",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToUtcDueOn(tt.dateStr, tt.timezone)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ToUtcDueOn(%q, %q) expected error, got nil", tt.dateStr, tt.timezone)
				}
				return
			}
			if err != nil {
				t.Fatalf("ToUtcDueOn(%q, %q) unexpected error: %v", tt.dateStr, tt.timezone, err)
			}
			if got != tt.want {
				t.Errorf("ToUtcDueOn(%q, %q) = %q, want %q", tt.dateStr, tt.timezone, got, tt.want)
			}
		})
	}
}
