package main

import (
	"testing"
	"time"
)

func TestFormat(t *testing.T) {
	for got, want := range map[string]string{
		bytesStr(512):               "512 B",
		bytesStr(3 << 20):           "3 MB",
		bytesStr(5 << 30):           "5.0 GB",
		rate(950):                   "950 B/s",
		rate(1500):                  "1.5 KB/s",
		rate(2.5e6):                 "2.5 MB/s",
		rate(4.2e7):                 "42 MB/s",
		gbAxis(1.3 * (1 << 30)):     "1.3 GB",
		gbAxis(48 * (1 << 30)):      "48 GB",
		gbAxis(256 << 20):           "256 MB",
		uptimeStr(90 * time.Minute): "1h 30m",
		uptimeStr(50 * time.Hour):   "2d 2h",
		pct(0.347):                  "35%",
	} {
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}
