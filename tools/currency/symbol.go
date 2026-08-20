package currency

import "strings"

// Symbol returns the display symbol for an ISO-4217 currency code
func Symbol(code string) string {

	switch strings.ToUpper(code) {

	case "USD":
		return "$"

	case "EUR":
		return "€"

	case "GBP":
		return "£"

	case "JPY":
		return "¥"

	case "AUD":
		return "A$"

	case "CAD":
		return "C$"

	case "CHF":
		return "CHF"

	case "CNY":
		return "¥"

	case "SEK":
		return "kr"

	case "NZD":
		return "$"
	default:
		return code
	}
}
