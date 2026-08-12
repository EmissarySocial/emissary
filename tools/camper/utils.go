package camper

import "strings"

// CanonicalCapitalization normalizes an intent name to FEP-3b86 form: first letter upper, rest lower
func CanonicalCapitalization(intent string) string {

	if intent == "" {
		return ""
	}

	return strings.ToUpper(intent[0:1]) + strings.ToLower(intent[1:])
}
