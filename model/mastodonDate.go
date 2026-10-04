package model

import "time"

// mastodonDateFormat is RFC3339 in UTC with mandatory millisecond precision.
// The official Mastodon iOS client's JSON date decoder only accepts
// ISO8601-with-fractional-seconds, a bare calendar date, or a numeric
// timestamp -- plain RFC3339 without the ".000" is rejected outright, which
// makes every Status/Account fail to decode and lists render empty. Do not
// switch to time.RFC3339Nano: it drops trailing zeros, so a whole-second time
// still serializes without a fractional part.
const mastodonDateFormat = "2006-01-02T15:04:05.000Z07:00"

// MastodonDate formats a time for the Mastodon API (see mastodonDateFormat).
func MastodonDate(t time.Time) string {
	return t.UTC().Format(mastodonDateFormat)
}
