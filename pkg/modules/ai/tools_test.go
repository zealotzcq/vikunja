package ai

import (
	"testing"
	"time"
)

func TestParseTimeExpression(t *testing.T) {
	now := time.Date(2024, 12, 20, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name      string
		expr      string
		wantErr   bool
		checkTime func(time.Time) bool
	}{
		{
			name:      "today",
			expr:      "today",
			wantErr:   false,
			checkTime: func(t time.Time) bool { return t.Equal(now) },
		},
		{
			name:      "tomorrow",
			expr:      "tomorrow",
			wantErr:   false,
			checkTime: func(t time.Time) bool { return t.Equal(now.AddDate(0, 0, 1)) },
		},
		{
			name:      "yesterday",
			expr:      "yesterday",
			wantErr:   false,
			checkTime: func(t time.Time) bool { return t.Equal(now.AddDate(0, 0, -1)) },
		},
		{
			name:      "今天",
			expr:      "今天",
			wantErr:   false,
			checkTime: func(t time.Time) bool { return t.Equal(now) },
		},
		{
			name:      "明天",
			expr:      "明天",
			wantErr:   false,
			checkTime: func(t time.Time) bool { return t.Equal(now.AddDate(0, 0, 1)) },
		},
		{
			name:      "2 hours",
			expr:      "in 2 hours",
			wantErr:   false,
			checkTime: func(t time.Time) bool { return t.Equal(now.Add(2 * time.Hour)) },
		},
		{
			name:      "30 minutes",
			expr:      "in 30 minutes",
			wantErr:   false,
			checkTime: func(t time.Time) bool { return t.Equal(now.Add(30 * time.Minute)) },
		},
		{
			name:      "2小时后",
			expr:      "2小时后",
			wantErr:   false,
			checkTime: func(t time.Time) bool { return t.Equal(now.Add(2 * time.Hour)) },
		},
		{
			name:      "30分钟后",
			expr:      "30分钟后",
			wantErr:   false,
			checkTime: func(t time.Time) bool { return t.Equal(now.Add(30 * time.Minute)) },
		},
		{
			name:      "3 days",
			expr:      "in 3 days",
			wantErr:   false,
			checkTime: func(t time.Time) bool { return t.Equal(now.AddDate(0, 0, 3)) },
		},
		{
			name:    "2024-12-25",
			expr:    "2024-12-25",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				return t.Year() == 2024 && t.Month() == 12 && t.Day() == 25
			},
		},
		{
			name:    "2024年12月25日",
			expr:    "2024年12月25日",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				return t.Year() == 2024 && t.Month() == 12 && t.Day() == 25
			},
		},
		{
			name:    "12月25日",
			expr:    "12月25日",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				return t.Month() == 12 && t.Day() == 25
			},
		},
		{
			name:    "next week",
			expr:    "next week",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				// Check if it's approximately next Sunday
				expected := time.Date(2024, 12, 22, 10, 30, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "下周",
			expr:    "下周",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				expected := time.Date(2024, 12, 22, 10, 30, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "本月",
			expr:    "本月",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				return t.Month() == 12 && t.Day() == 31
			},
		},
		{
			name:      "invalid",
			expr:      "invalid time",
			wantErr:   true,
			checkTime: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTimeExpression(tt.expr, now)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTimeExpression() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !tt.checkTime(got) {
				t.Errorf("parseTimeExpression() = %v, check failed", got)
			}
		})
	}
}
