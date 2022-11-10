package tinyDate

import (
	"strconv"
	"time"

	"github.com/davecgh/go-spew/spew"
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
	days := hours / 24
	if days < 30 {
		return strconv.Itoa(int(days)) + "d"
	}

	months := ((secondTime.Year() - firstTime.Year()) * 12) + (int(secondTime.Month()) - int(firstTime.Month()))
	spew.Dump(months)
	if months < 12 {
		return strconv.Itoa(int(months)) + "mo"
	}

	years := months / 12
	spew.Dump(years)
	return strconv.Itoa(int(years)) + "y"
}
