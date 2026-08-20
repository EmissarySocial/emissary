package convert

import "github.com/microcosm-cc/bluemonday"

// SanitizeHTML removes unsafe markup from a string, leaving user-generated HTML intact
func SanitizeHTML(value string) string {
	return bluemonday.UGCPolicy().Sanitize(value)
}

// SanitizeText removes all markup from a string, leaving plain text
func SanitizeText(value string) string {
	return bluemonday.StrictPolicy().Sanitize(value)
}
