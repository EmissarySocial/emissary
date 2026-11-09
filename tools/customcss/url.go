package customcss

import (
	"strings"
)

// isLocalURL reports whether a browser would resolve an address to the page's own website, or to a data: URI
func isLocalURL(value string) bool {

	// A browser's URL parser drops every tab and newline, and C0 controls and spaces at either end
	value = strings.Map(dropTabsAndNewlines, value)
	value = strings.TrimFunc(value, isControlOrSpace)

	// ...and reads a backslash as a slash in an http(s) address, so "/\host" means "//host"
	value = strings.ReplaceAll(value, `\`, "/")

	// RULE: "//host/path" names another host
	if strings.HasPrefix(value, "//") {
		return false
	}

	// RULE: An address with a scheme may only be a data: URI, which loads nothing
	if scheme, hasScheme := urlScheme(value); hasScheme {
		return strings.EqualFold(scheme, "data")
	}

	// Everything else is a path on this website
	return true
}

// urlScheme returns the scheme that begins an address, and FALSE when the address is relative
func urlScheme(value string) (string, bool) {

	for index := 0; index < len(value); index++ {

		switch character := value[index]; {

		case character == ':':
			return value[:index], (index > 0)

		case isASCIILetter(character):
			continue

		case index == 0:
			return "", false

		case (character >= '0') && (character <= '9'):
			continue

		case (character == '+') || (character == '-') || (character == '.'):
			continue
		}

		return "", false
	}

	return "", false
}

// isASCIILetter reports whether a byte is an ASCII letter
func isASCIILetter(character byte) bool {
	return ((character >= 'a') && (character <= 'z')) || ((character >= 'A') && (character <= 'Z'))
}

// dropTabsAndNewlines removes the characters that a browser's URL parser removes everywhere
func dropTabsAndNewlines(character rune) rune {

	switch character {

	case '\t', '\n', '\r':
		return -1
	}

	return character
}

// isControlOrSpace reports whether a character is one that a browser's URL parser trims from either end
func isControlOrSpace(character rune) bool {
	return character <= ' '
}
