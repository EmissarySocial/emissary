package camper

import "strings"

func CanonicalCapitalization(intent string) string {

	if intent == "" {
		return ""
	}

	return strings.ToUpper(intent[0:1]) + strings.ToLower(intent[1:])
}
