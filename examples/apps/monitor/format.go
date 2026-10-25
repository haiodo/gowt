package main

import (
	"fmt"
	"math"
	"time"
)

func pct(v float64) string { return fmt.Sprintf("%.0f%%", v*100) }

func bytesStr(n uint64) string {
	const k = 1024.0
	v := float64(n)
	switch {
	case v >= k*k*k*k:
		return fmt.Sprintf("%.1f TB", v/(k*k*k*k))
	case v >= k*k*k:
		return fmt.Sprintf("%.1f GB", v/(k*k*k))
	case v >= k*k:
		return fmt.Sprintf("%.0f MB", v/(k*k))
	case v >= k:
		return fmt.Sprintf("%.0f KB", v/k)
	}
	return fmt.Sprintf("%d B", int64(n))
}

// rate formats bytes per second; the unit steps at 1000 like network tools do.
func rate(v float64) string {
	switch {
	case v >= 1e9:
		return fmt.Sprintf("%.1f GB/s", v/1e9)
	case v >= 1e7:
		return fmt.Sprintf("%.0f MB/s", v/1e6)
	case v >= 1e6:
		return fmt.Sprintf("%.1f MB/s", v/1e6)
	case v >= 1e4:
		return fmt.Sprintf("%.0f KB/s", v/1e3)
	case v >= 1e3:
		return fmt.Sprintf("%.1f KB/s", v/1e3)
	}
	return fmt.Sprintf("%.0f B/s", v)
}

func uptimeStr(d time.Duration) string {
	days, hours, mins := int(d.Hours())/24, int(d.Hours())%24, int(d.Minutes())%60
	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh", days, hours)
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

// gbAxis labels a memory gridline: whole GB from 10 GB up, one decimal below, so a 5.3 GB machine
// reads 1.3, 2.7, 4.0 and 5.3 rather than a rounded figure that is not where the line is.
func gbAxis(b float64) string {
	gb := b / (1 << 30)
	switch {
	case gb >= 10:
		return fmt.Sprintf("%.0f GB", math.Round(gb))
	case gb >= 0.95:
		return fmt.Sprintf("%.1f GB", gb)
	}
	return fmt.Sprintf("%.0f MB", math.Round(b/(1<<20)/16)*16)
}
