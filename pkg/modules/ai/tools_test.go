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
		{
			name:    "Tuesday in next week",
			expr:    "Tuesday in next week",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				expected := time.Date(2024, 12, 24, 10, 30, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "周五在下周",
			expr:    "周五在下周",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				expected := time.Date(2024, 12, 27, 10, 30, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "下周三",
			expr:    "下周三",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				expected := time.Date(2024, 12, 25, 10, 30, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "25th day in next month",
			expr:    "25th day in next month",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				expected := time.Date(2025, 1, 25, 10, 30, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "下个月的25号",
			expr:    "下个月的25号",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				expected := time.Date(2025, 1, 25, 10, 30, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "15th day in this month",
			expr:    "15th day in this month",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				expected := time.Date(2024, 12, 15, 10, 30, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "本月15号",
			expr:    "本月15号",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				expected := time.Date(2024, 12, 15, 10, 30, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "Friday in this week",
			expr:    "Friday in this week",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				// Dec 20, 2024 is Friday, so Friday of this week is today
				expected := time.Date(2024, 12, 20, 10, 30, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "Monday in this week",
			expr:    "Monday in this week",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				// Dec 20, 2024 is Friday, Monday of this week was Dec 16
				// But we should return the next Monday which is Dec 23
				expected := time.Date(2024, 12, 23, 10, 30, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "Wednesday in last week",
			expr:    "Wednesday in last week",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				// Dec 20, 2024 is Friday, Wednesday of last week was Dec 11
				expected := time.Date(2024, 12, 11, 10, 30, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "last week Friday",
			expr:    "last week Friday",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				// Dec 20, 2024 is Friday, Friday of last week was Dec 13
				expected := time.Date(2024, 12, 13, 10, 30, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "10th day in last month",
			expr:    "10th day in last month",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				// Dec 20, 2024, last month is November
				expected := time.Date(2024, 11, 10, 10, 30, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "上月10号",
			expr:    "上月10号",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				expected := time.Date(2024, 11, 10, 10, 30, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "8am",
			expr:    "8am",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				expected := time.Date(2024, 12, 20, 8, 0, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "3:30pm",
			expr:    "3:30pm",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				expected := time.Date(2024, 12, 20, 15, 30, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "11:00 am",
			expr:    "11:00 am",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				expected := time.Date(2024, 12, 20, 11, 0, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "14:00",
			expr:    "14:00",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				expected := time.Date(2024, 12, 20, 14, 0, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "9:30",
			expr:    "9:30",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				expected := time.Date(2024, 12, 20, 9, 30, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "tomorrow at 8am",
			expr:    "tomorrow at 8am",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				expected := time.Date(2024, 12, 21, 8, 0, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "next Monday at 3pm",
			expr:    "next Monday at 3pm",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				// Dec 20, 2024 is Friday, next Monday is Dec 23
				expected := time.Date(2024, 12, 23, 15, 0, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "tomorrow at 9:30am",
			expr:    "tomorrow at 9:30am",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				expected := time.Date(2024, 12, 21, 9, 30, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "today at 5pm",
			expr:    "today at 5pm",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				expected := time.Date(2024, 12, 20, 17, 0, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "next week at 10am",
			expr:    "next week at 10am",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				// Dec 20, 2024 is Friday, next week starts Sunday Dec 22 at 10am
				expected := time.Date(2024, 12, 22, 10, 0, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "下周三 at 2pm",
			expr:    "下周三 at 2pm",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				expected := time.Date(2024, 12, 25, 14, 0, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "12:00am",
			expr:    "12:00am",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				expected := time.Date(2024, 12, 20, 0, 0, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "12:00pm",
			expr:    "12:00pm",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				expected := time.Date(2024, 12, 20, 12, 0, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "tomorrow at invalid",
			expr:    "tomorrow at invalid",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				// Fallback: should return tomorrow at current time
				expected := time.Date(2024, 12, 21, 10, 30, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "next Monday at abc",
			expr:    "next Monday at abc",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				// Fallback: should return next Monday at current time
				// Dec 20, 2024 is Friday, next Monday is Dec 23
				expected := time.Date(2024, 12, 23, 10, 30, 0, 0, time.UTC)
				return t.Equal(expected)
			},
		},
		{
			name:    "today at 25:00",
			expr:    "today at 25:00",
			wantErr: false,
			checkTime: func(t time.Time) bool {
				// Fallback: invalid time, should return today at current time
				expected := time.Date(2024, 12, 20, 10, 30, 0, 0, time.UTC)
				return t.Equal(expected)
			},
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
