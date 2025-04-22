package dateparse

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantDate   time.Time
		wantIsWeek bool
		nowFunc    func() time.Time
		expectErr  bool
	}{
		{
			name:       "No arguments",
			args:       []string{},
			wantDate:   time.Now().Truncate(24 * time.Hour),
			wantIsWeek: false,
			expectErr:  false,
		},
		{
			name:       "Only username",
			args:       []string{"alice"},
			wantDate:   time.Now().Truncate(24 * time.Hour),
			wantIsWeek: false,
			expectErr:  false,
		},
		{
			name:       "Username and tomorrow",
			args:       []string{"bob", "tomorrow"},
			wantDate:   time.Now().Truncate(24 * time.Hour).Add(24 * time.Hour),
			wantIsWeek: false,
			expectErr:  false,
		},
		{
			name:       "Username and specific date",
			args:       []string{"carol", "2025-12-25"},
			wantDate:   time.Date(2025, 12, 25, 0, 0, 0, 0, time.Local),
			wantIsWeek: false,
			expectErr:  false,
		},
		{
			name:       "Invalid date format",
			args:       []string{"dave", "invalid-date"},
			wantDate:   time.Now().Truncate(24 * time.Hour),
			wantIsWeek: false,
			expectErr:  false, // Even with invalid date, it defaults to today
		},
		{
			name: "Username and next day of week",
			args: []string{"eve", "next", "monday"},
			// use a custom now function to ensure the test is deterministic.
			// always return 2025-01-31
			nowFunc: func() time.Time {
				return time.Date(2025, 1, 31, 0, 0, 0, 0, time.Local)
			},
			wantDate:   time.Date(2025, 2, 3, 0, 0, 0, 0, time.Local),
			wantIsWeek: false,
			expectErr:  false,
		},
		{
			name: "Username and week",
			args: []string{"frank", "week"},
			nowFunc: func() time.Time {
				return time.Date(2025, 1, 31, 0, 0, 0, 0, time.Local) // Friday
			},
			wantDate:   time.Date(2025, 1, 31, 0, 0, 0, 0, time.Local),
			wantIsWeek: true,
			expectErr:  false,
		},
		{
			name: "Username and next week",
			args: []string{"grace", "next", "week"},
			nowFunc: func() time.Time {
				return time.Date(2025, 1, 31, 0, 0, 0, 0, time.Local) // Friday
			},
			wantDate:   time.Date(2025, 1, 31, 0, 0, 0, 0, time.Local),
			wantIsWeek: true,
			expectErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := New()
			if tt.nowFunc != nil {
				parser.NowDate = tt.nowFunc
			}
			result, err := parser.Parse(tt.args)
			if (err != nil) != tt.expectErr {
				t.Errorf("Parse() error = %v, expectErr %v", err, tt.expectErr)
				return
			}

			if !tt.expectErr {
				// Check if IsWeek flag is set correctly
				if result.IsWeek != tt.wantIsWeek {
					t.Errorf("Parse() IsWeek = %v, want %v", result.IsWeek, tt.wantIsWeek)
				}

				// For week requests, check if WeekDays is populated
				if tt.wantIsWeek {
					if len(result.WeekDays) != 7 {
						t.Errorf("Parse() WeekDays length = %v, want 7", len(result.WeekDays))
					}

					// For "week", the first day should be a Monday
					if result.WeekDays[0].Weekday() != time.Monday {
						t.Errorf("Parse() first day of week = %v, want Monday", result.WeekDays[0].Weekday())
					}

					// For "next week", the first day should be the Monday after the current date
					if tt.args[1] == "next" && tt.args[2] == "week" {
						// The Monday should be after the current date
						if !result.WeekDays[0].After(tt.nowFunc()) {
							t.Errorf("Parse() next week Monday = %v, should be after current date %v",
								result.WeekDays[0], tt.nowFunc())
						}
					}
				} else {
					// For non-week requests, check the date
					if !tt.wantDate.IsZero() {
						// Compare dates ignoring the exact time of execution
						if result.Date.Year() != tt.wantDate.Year() ||
							result.Date.Month() != tt.wantDate.Month() ||
							result.Date.Day() != tt.wantDate.Day() {
							t.Errorf("Parse() date = %v, want %v", result.Date, tt.wantDate)
						}
					}
				}
			}
		})
	}
}
