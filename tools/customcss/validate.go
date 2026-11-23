package customcss

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/benpate/derp"
	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

// Validate returns nil when a stylesheet is safe to place in a <style> element on a page that
// other people view, or a validation error naming the first rule that it breaks
func Validate(value string) error {
	return validate(value, false)
}

// validate applies every rule to a stylesheet, including the rule against files from other
// websites unless remoteAllowed
func validate(value string, remoteAllowed bool) error {

	// An empty stylesheet has nothing to check
	if value == "" {
		return nil
	}

	// RULE: Bound the work before reading anything
	if len(value) > maxLength {
		return derp.Validation(fmt.Sprintf("The stylesheet is longer than %d bytes.", maxLength))
	}

	// RULE: The lexer reads UTF-8, and treats a NUL byte as the end of its input,
	// which would leave everything after it unchecked
	if !utf8.ValidString(value) {
		return derp.Validation("The stylesheet is not valid UTF-8.")
	}

	if index := strings.IndexByte(value, 0); index >= 0 {
		return failureAt(value, index, "contains a null character")
	}

	// RULE: "</" could close the <style> element, wherever it appears
	if index := strings.Index(value, "</"); index >= 0 {
		return failureAt(value, index, `cannot contain "</"`)
	}

	// RULE: A final, unescaped backslash would escape whatever the page writes after the stylesheet
	if endsWithLoneBackslash(value) {
		return failureAt(value, len(value)-1, "ends with a backslash")
	}

	// Check every token, in the order a browser reads them
	walker := newWalker(remoteAllowed)
	lexer := css.NewLexer(parse.NewInputString(value))

	for {
		tokenType, data := lexer.Next()

		if tokenType == css.ErrorToken {
			break
		}

		if err := walker.token(tokenType, data); err != nil {
			return err
		}
	}

	// RULE: The lexer must have read the whole stylesheet. UNREACHABLE by test: with NUL
	// refused above, the lexer stops only at the end of its input, having read every byte.
	if !errors.Is(lexer.Err(), io.EOF) {
		return derp.Validation("The stylesheet could not be read to the end.")
	}

	if walker.consumed != len(value) {
		return derp.Validation("The stylesheet could not be read to the end.")
	}

	// RULE: Every block must be closed
	return walker.finish()
}

// endsWithLoneBackslash reports whether a stylesheet ends with an odd run of backslashes
func endsWithLoneBackslash(value string) bool {
	trimmed := strings.TrimRight(value, `\`)
	return ((len(value) - len(trimmed)) % 2) == 1
}

// failureAt returns a validation error for a rule broken at a byte offset in the stylesheet
func failureAt(value string, index int, message string) error {
	line := 1 + strings.Count(value[:index], "\n")
	return derp.Validation(fmt.Sprintf("The stylesheet %s (line %d).", message, line))
}
