package main

import (
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	cases := []struct {
		name string
		in   string
		ok   bool
	}{
		{"default", "27.03.2014", true},
		{"leap day", "29.02.2012", true},
		{"not a leap day", "29.02.2013", false},
		{"iso order", "2014-03-27", false},
		{"empty", "", false},
		{"trailing space", "27.03.2014 ", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseDate(tc.in)
			if ok := err == nil; ok != tc.ok {
				t.Fatalf("parseDate(%q) error = %v, want success %v", tc.in, err, tc.ok)
			}
		})
	}
}

func TestBookable(t *testing.T) {
	first := time.Date(2014, 3, 27, 0, 0, 0, 0, time.UTC)
	later := time.Date(2014, 4, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		mode int
		a, b time.Time
		want bool
	}{
		{"one-way ignores the dates", oneWay, later, first, true},
		{"return on the same day", returnFlight, first, first, true},
		{"return in order", returnFlight, first, later, true},
		{"return out of order", returnFlight, later, first, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := bookable(tc.mode, tc.a, tc.b); got != tc.want {
				t.Fatalf("bookable(%d, %v, %v) = %v, want %v", tc.mode, tc.a, tc.b, got, tc.want)
			}
		})
	}
}
