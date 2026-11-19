package customcss

import (
	"bytes"
	"strings"
	"unicode"
	"unicode/utf8"
)

// identifier returns the name in an identifier, function, or at-keyword token, unescaped and lowercased
func identifier(data []byte) string {
	return strings.ToLower(unescape(string(data)))
}

// unescape replaces every CSS escape in a token with the character that it stands for
func unescape(value string) string {

	if !strings.Contains(value, `\`) {
		return value
	}

	var result strings.Builder
	result.Grow(len(value))

	for index := 0; index < len(value); {

		// Everything except a backslash stands for itself
		if value[index] != '\\' {
			result.WriteByte(value[index])
			index++
			continue
		}

		index++

		// A backslash at the very end stands for nothing
		if index >= len(value) {
			break
		}

		// A backslash before a newline continues a string onto the next line
		if width := newlineWidth(value[index:]); width > 0 {
			index += width
			continue
		}

		// Up to six hex digits name a code point, and one whitespace after them belongs to the escape
		if digits := hexDigits(value[index:]); digits > 0 {
			result.WriteRune(codePoint(value[index : index+digits]))
			index += digits
			index += whitespaceWidth(value[index:])
			continue
		}

		// Any other character after a backslash stands for itself
		character, width := utf8.DecodeRuneInString(value[index:])
		result.WriteRune(character)
		index += width
	}

	return result.String()
}

// hexDigits returns how many hex digits, at most six, begin a string
func hexDigits(value string) int {

	count := 0

	for (count < len(value)) && (count < 6) && isHexDigit(value[count]) {
		count++
	}

	return count
}

// isHexDigit reports whether a byte is a hexadecimal digit
func isHexDigit(character byte) bool {

	switch {

	case (character >= '0') && (character <= '9'):
		return true

	case (character >= 'a') && (character <= 'f'):
		return true

	case (character >= 'A') && (character <= 'F'):
		return true
	}

	return false
}

// codePoint reads one to six hex digits as a code point, replacing any that CSS forbids
func codePoint(digits string) rune {

	value := 0

	for index := 0; index < len(digits); index++ {
		value = (value * 16) + hexValue(digits[index])
	}

	// RULE: NUL, surrogates, and values past the last code point become U+FFFD, as in a browser
	if (value == 0) || (value > unicode.MaxRune) || ((value >= 0xD800) && (value <= 0xDFFF)) {
		return utf8.RuneError
	}

	return rune(value)
}

// hexValue returns the value of one hexadecimal digit
func hexValue(character byte) int {

	switch {

	case character <= '9':
		return int(character - '0')

	case character <= 'F':
		return int(character-'A') + 10
	}

	return int(character-'a') + 10
}

// newlineWidth returns the length of the newline that begins a string, or zero when there is none
func newlineWidth(value string) int {

	switch {

	case strings.HasPrefix(value, "\r\n"):
		return 2

	case strings.HasPrefix(value, "\n"), strings.HasPrefix(value, "\r"), strings.HasPrefix(value, "\f"):
		return 1
	}

	return 0
}

// whitespaceWidth returns the length of the one whitespace character that begins a string, or zero
func whitespaceWidth(value string) int {

	if width := newlineWidth(value); width > 0 {
		return width
	}

	if strings.HasPrefix(value, " ") || strings.HasPrefix(value, "\t") {
		return 1
	}

	return 0
}

// isTerminatedString reports whether a quoted string token ends with an unescaped copy of its opening quote
func isTerminatedString(data []byte) bool {

	if len(data) < 2 {
		return false
	}

	if data[len(data)-1] != data[0] {
		return false
	}

	// An odd run of backslashes before the final quote escapes it
	backslashes := 0

	for index := len(data) - 2; (index > 0) && (data[index] == '\\'); index-- {
		backslashes++
	}

	return (backslashes % 2) == 0
}

// isTerminatedComment reports whether a comment token ends with "*/"
func isTerminatedComment(data []byte) bool {
	return (len(data) >= 4) && bytes.HasSuffix(data, []byte("*/"))
}

// urlValue returns the unescaped address inside a url() token, and FALSE when the token never closes
func urlValue(data []byte) (string, bool) {

	// UNREACHABLE by test: the lexer emits a URL token only after reading "url("
	_, inside, found := strings.Cut(string(data), "(")

	if !found {
		return "", false
	}

	// The lexer ends a url() at the end of its input, whether or not it was closed,
	// and an escaped ")" does not close it
	inside, isClosed := strings.CutSuffix(inside, ")")

	if !isClosed {
		return "", false
	}

	if endsWithLoneBackslash(inside) {
		return "", false
	}

	inside = strings.Trim(inside, cssWhitespace)

	if inside == "" {
		return "", true
	}

	// An unquoted address is unescaped as it stands
	if (inside[0] != '"') && (inside[0] != '\'') {
		return unescape(inside), true
	}

	// A quoted address must close its quote before the url() closes
	if !isTerminatedString([]byte(inside)) {
		return "", false
	}

	return unescape(inside[1 : len(inside)-1]), true
}
