package dateparse

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantUser  string
		wantDate  time.Time
		nowFunc   func() time.Time
		expectErr bool
	}{
		{
			name:      "No arguments",
			args:      []string{},
			wantUser:  "",
			expectErr: true,
		},
		{
			name:      "Only username",
			args:      []string{"alice"},
			wantUser:  "alice",
			wantDate:  time.Now().Truncate(24 * time.Hour),
			expectErr: false,
		},
		{
			name:      "Username and tomorrow",
			args:      []string{"bob", "tomorrow"},
			wantUser:  "bob",
			wantDate:  time.Now().Truncate(24 * time.Hour).Add(24 * time.Hour),
			expectErr: false,
		},
		{
			name:      "Username and specific date",
			args:      []string{"carol", "2025-12-25"},
			wantUser:  "carol",
			wantDate:  time.Date(2025, 12, 25, 0, 0, 0, 0, time.Local),
			expectErr: false,
		},
		{
			name:      "Invalid date format",
			args:      []string{"dave", "invalid-date"},
			wantUser:  "dave",
			wantDate:  time.Now().Truncate(24 * time.Hour),
			expectErr: false, // Even with invalid date, it defaults to today
		},
		{
			name:     "Username and next day of week",
			args:     []string{"eve", "next", "monday"},
			wantUser: "eve",
			// use a custom now function to ensure the test is deterministic.
			// always return 2025-01-31
			nowFunc: func() time.Time {
				return time.Date(2025, 1, 31, 0, 0, 0, 0, time.Local)
			},
			wantDate:  time.Date(2025, 2, 3, 0, 0, 0, 0, time.Local),
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := New()
			if tt.nowFunc != nil {
				parser.NowDate = tt.nowFunc
			}
			user, date, err := parser.Parse(tt.args)
			if (err != nil) != tt.expectErr {
				t.Errorf("Parse() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if user != tt.wantUser && !tt.expectErr {
				t.Errorf("Parse() user = %v, want %v", user, tt.wantUser)
			}
			if !tt.wantDate.IsZero() && !tt.expectErr {
				// Compare dates ignoring the exact time of execution
				if date.Year() != tt.wantDate.Year() ||
					date.Month() != tt.wantDate.Month() ||
					date.Day() != tt.wantDate.Day() {
					t.Errorf("Parse() date = %v, want %v", date, tt.wantDate)
				}
			}
		})
	}
}
