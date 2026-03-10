package milestone

import (
	"strings"
	"testing"
	"time"
)

func TestNextMonday(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		now  time.Time
		want string
	}{
		{
			name: "Wednesday returns next Monday",
			now:  time.Date(2026, 3, 11, 12, 0, 0, 0, time.UTC), // Wednesday
			want: "2026-03-16",
		},
		{
			name: "Monday returns following Monday",
			now:  time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC), // Monday
			want: "2026-03-23",
		},
		{
			name: "Sunday returns next day Monday",
			now:  time.Date(2026, 3, 15, 23, 59, 0, 0, time.UTC), // Sunday
			want: "2026-03-16",
		},
		{
			name: "Saturday returns Monday in 2 days",
			now:  time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC), // Saturday
			want: "2026-03-16",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NextMonday(tt.now)
			if got != tt.want {
				t.Errorf("NextMonday(%v) = %q, want %q", tt.now.Format("2006-01-02"), got, tt.want)
			}
		})
	}
}

func TestWeeksUntilEndOfYear(t *testing.T) {
	t.Parallel()
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
		{
			name:      "start at Dec 31 returns 1",
			startDate: "2026-12-31",
			want:      1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := WeeksUntilEndOfYear(tt.startDate)
			if err != nil {
				t.Fatalf("WeeksUntilEndOfYear(%q) unexpected error: %v", tt.startDate, err)
			}
			if got != tt.want {
				t.Errorf("WeeksUntilEndOfYear(%q) = %d, want %d", tt.startDate, got, tt.want)
			}
		})
	}

	t.Run("invalid date returns error", func(t *testing.T) {
		t.Parallel()
		_, err := WeeksUntilEndOfYear("not-a-date")
		if err == nil {
			t.Fatal("WeeksUntilEndOfYear(\"not-a-date\") expected error, got nil")
		}
		if !strings.Contains(err.Error(), "invalid start date") {
			t.Errorf("error = %q, want it to contain %q", err, "invalid start date")
		}
	})
}

func TestISOWeek(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			got := ISOWeek(tt.date)
			if got != tt.want {
				t.Errorf("ISOWeek(%v) = %d, want %d", tt.date.Format("2006-01-02"), got, tt.want)
			}
		})
	}
}

func TestToUtcDueOn(t *testing.T) {
	t.Parallel()
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
			name:     "America/New_York UTC-4 summer DST",
			dateStr:  "2026-07-04",
			timezone: "America/New_York",
			want:     "2026-07-05T03:59:59Z",
		},
		{
			name:     "empty timezone treated as UTC",
			dateStr:  "2026-01-04",
			timezone: "",
			want:     "2026-01-04T23:59:59Z",
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
			t.Parallel()
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
