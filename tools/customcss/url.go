package customcss

import (
	"net/url"
	"strings"
)

// fileRoutes are the only paths beginning with a dot that serve files
var fileRoutes = []string{".themes", ".templates", ".widgets", ".domain", ".searchTag"}

// isServerRoute reports whether a local address reaches one of the server's own dot-prefixed routes,
// such as /.intents, by any spelling.  Only the routes that serve files are left alone.
func isServerRoute(value string) bool {

	// Read the address the way isLocalURL does, then keep only its path
	value = strings.Map(dropTabsAndNewlines, value)
	value = strings.TrimFunc(value, isControlOrSpace)
	value = strings.ReplaceAll(value, `\`, "/")

	// RULE: A data: URI is not a path, and loads nothing
	if _, hasScheme := urlScheme(value); hasScheme {
		return false
	}

	path, _, _ := strings.Cut(value, "#")
	path, _, _ = strings.Cut(path, "?")
	isAbsolute := strings.HasPrefix(path, "/")

	for index, segment := range strings.Split(strings.TrimPrefix(path, "/"), "/") {

		// RULE: A segment that cannot be decoded is refused, because the router may read it differently
		decoded, err := url.PathUnescape(segment)

		if err != nil {
			return true
		}

		if !strings.HasPrefix(decoded, ".") {
			continue
		}

		// "." and ".." only move between folders
		if (decoded == ".") || (decoded == "..") {
			continue
		}

		// RULE: A file route counts only as the first segment of an absolute path
		if isAbsolute && (index == 0) && isFileRoute(decoded) {
			continue
		}

		return true
	}

	return false
}

// isFileRoute reports whether a dot-prefixed first segment names a route that serves files
func isFileRoute(segment string) bool {

	for _, fileRoute := range fileRoutes {
		if segment == fileRoute {
			return true
		}
	}

	return false
}

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
