// Package tinyDate provides a nifty way to format dates, just like those fancy tech-bro's do in Silicon Valley.
package tinyDate

import (
	"strconv"
	"time"
)

// FormaDiff returns a string representation of the duration since this date in as little space as possible.
func FormatDiff(firstTime time.Time, secondTime time.Time) string {
	seconds := secondTime.Unix() - firstTime.Unix()

	// If seconds, say "just now"
	if seconds < 60 {
		return strconv.Itoa(int(seconds)) + "s"
	}

	// If minutes, say "1min"
	minutes := seconds / 60

	if minutes < 60 {
		return strconv.Itoa(int(minutes)) + "min"
	}

	// If hours, say "1h"
	hours := minutes / 60
	if hours < 24 {
		return strconv.Itoa(int(hours)) + "h"
	}

	// If days, say "1d"
	if days := hours / 24; days < 30 {
		return strconv.Itoa(int(days)) + "d"
	}

	// If months, say "1mo"
	months := ((secondTime.Year() - firstTime.Year()) * 12) + (int(secondTime.Month()) - int(firstTime.Month()))
	if months < 12 {
		return strconv.Itoa(int(months)) + "mo"
	}

	// Otherwise, it's years.  Say "1y"
	years := months / 12
	return strconv.Itoa(int(years)) + "y"
}
