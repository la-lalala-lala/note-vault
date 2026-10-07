package service

import (
	"testing"
	"time"
)

func TestCurrentWeekRange(t *testing.T) {
	location := time.FixedZone("CST", 8*60*60)
	tests := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{name: "周三", now: time.Date(2026, time.October, 7, 15, 30, 0, 0, location), want: time.Date(2026, time.October, 5, 0, 0, 0, 0, location)},
		{name: "周日", now: time.Date(2026, time.October, 11, 23, 59, 0, 0, location), want: time.Date(2026, time.October, 5, 0, 0, 0, 0, location)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			start, end := currentWeekRange(test.now)
			if !start.Equal(test.want) {
				t.Fatalf("周起始时间 = %s，期望 %s", start, test.want)
			}
			if !end.Equal(test.want.AddDate(0, 0, 7)) {
				t.Fatalf("周结束时间 = %s，期望 %s", end, test.want.AddDate(0, 0, 7))
			}
		})
	}
}
