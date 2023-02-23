package convert

import "github.com/microcosm-cc/bluemonday"

func SanitizeHTML(value string) string {
	return bluemonday.UGCPolicy().Sanitize(value)
}

func SanitizeText(value string) string {
	return bluemonday.StrictPolicy().Sanitize(value)
}
