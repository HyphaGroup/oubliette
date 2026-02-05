package schedule

import (
	"testing"
	"time"
)

func TestParseCron_Valid(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{"every minute", "* * * * *"},
		{"hourly", "0 * * * *"},
		{"daily at midnight", "0 0 * * *"},
		{"weekly on Sunday", "0 0 * * 0"},
		{"monthly on 1st", "0 0 1 * *"},
		{"every 5 minutes", "*/5 * * * *"},
		{"workday 9am", "0 9 * * 1-5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseCron(tt.expr)
			if err != nil {
				t.Errorf("ParseCron(%q) error = %v, want nil", tt.expr, err)
			}
		})
	}
}

func TestParseCron_Invalid(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{"empty", ""},
		{"too few fields", "* * *"},
		{"too many fields", "* * * * * *"},
		{"invalid minute", "60 * * * *"},
		{"invalid hour", "* 25 * * *"},
		{"invalid day", "* * 32 * *"},
		{"invalid month", "* * * 13 *"},
		{"invalid weekday", "* * * * 8"},
		{"garbage", "not a cron"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseCron(tt.expr)
			if err == nil {
				t.Errorf("ParseCron(%q) error = nil, want error", tt.expr)
			}
		})
	}
}

func TestNextRun(t *testing.T) {
	// Test at a specific time
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		expr     string
		after    time.Time
		wantYear int
		wantMon  time.Month
		wantDay  int
		wantHour int
		wantMin  int
	}{
		{
			name:     "next minute",
			expr:     "* * * * *",
			after:    now,
			wantYear: 2025, wantMon: 1, wantDay: 15, wantHour: 10, wantMin: 31,
		},
		{
			name:     "next hour",
			expr:     "0 * * * *",
			after:    now,
			wantYear: 2025, wantMon: 1, wantDay: 15, wantHour: 11, wantMin: 0,
		},
		{
			name:     "next day at midnight",
			expr:     "0 0 * * *",
			after:    now,
			wantYear: 2025, wantMon: 1, wantDay: 16, wantHour: 0, wantMin: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next, err := NextRun(tt.expr, tt.after)
			if err != nil {
				t.Fatalf("NextRun(%q, %v) error = %v", tt.expr, tt.after, err)
			}

			if next.Year() != tt.wantYear || next.Month() != tt.wantMon || next.Day() != tt.wantDay ||
				next.Hour() != tt.wantHour || next.Minute() != tt.wantMin {
				t.Errorf("NextRun(%q, %v) = %v, want %d-%02d-%02d %02d:%02d",
					tt.expr, tt.after, next,
					tt.wantYear, tt.wantMon, tt.wantDay, tt.wantHour, tt.wantMin)
			}
		})
	}
}

func TestNextRun_InvalidCron(t *testing.T) {
	_, err := NextRun("invalid cron", time.Now())
	if err == nil {
		t.Error("NextRun with invalid cron should return error")
	}
}

func TestValidateCron(t *testing.T) {
	if err := ValidateCron("0 0 * * *"); err != nil {
		t.Errorf("ValidateCron for valid cron returned error: %v", err)
	}

	if err := ValidateCron("invalid"); err == nil {
		t.Error("ValidateCron for invalid cron should return error")
	}
}
