package customcss

import (
	"bytes"
	"fmt"

	"github.com/benpate/derp"
	"github.com/tdewolff/parse/v2/css"
)

// walker follows a stylesheet's tokens, tracking the blocks that are open and the current line
type walker struct {
	remoteAllowed bool
	blocks        []block
	line          int
	consumed      int
}

// block is one open bracket, the line it opened on, and the name of its function, if it has one
type block struct {
	opener   byte
	closer   byte
	line     int
	function string
}

// newWalker returns a walker positioned at the start of a stylesheet
func newWalker(remoteAllowed bool) walker {
	return walker{
		remoteAllowed: remoteAllowed,
		blocks:        make([]block, 0, 8),
		line:          1,
	}
}

// token checks one token against every rule, then moves past it
func (w *walker) token(tokenType css.TokenType, data []byte) error {

	if err := w.check(tokenType, data); err != nil {
		return err
	}

	w.consumed += len(data)
	w.line += bytes.Count(data, []byte{'\n'})
	return nil
}

// check applies the rules that concern one kind of token
func (w *walker) check(tokenType css.TokenType, data []byte) error {

	// UNREACHABLE by test: the lexer returns an empty token only with ErrorToken, which ends the walk
	if len(data) == 0 {
		return nil
	}

	switch tokenType {

	case css.LeftBraceToken:
		w.open('{', '}', "")

	case css.LeftBracketToken:
		w.open('[', ']', "")

	case css.LeftParenthesisToken:
		w.open('(', ')', "")

	case css.FunctionToken:
		name := identifier(bytes.TrimSuffix(data, []byte("(")))

		// RULE: A browser reads an escaped "url(" as url(), but the lexer reads it as an ordinary function
		if name == "url" {
			return w.failure("spells url( with escape codes")
		}

		// RULE: A substituted value inside image-set() and its relatives is an address nobody checked
		if w.substitutesAddress(name) {
			return w.failure("cannot use " + name + "() inside a function that loads a file")
		}

		w.open('(', ')', name)

	case css.RightBraceToken, css.RightBracketToken, css.RightParenthesisToken:
		return w.close(data[0])

	case css.AtKeywordToken:

		// RULE: @import loads a whole stylesheet from wherever it names
		if identifier(bytes.TrimPrefix(data, []byte("@"))) == "import" {
			return w.checkImport()
		}

	case css.StringToken:

		if !isTerminatedString(data) {
			return w.failure("has a string that never ends")
		}

		// A string inside image-set() and its relatives is an address to load
		if w.inURLFunction() {
			return w.checkURL(unescape(string(data[1 : len(data)-1])))
		}

	case css.BadStringToken:
		return w.failure("has a string that runs onto a new line")

	case css.URLToken:

		value, isTerminated := urlValue(data)

		if !isTerminated {
			return w.failure("has a url() that never ends")
		}

		return w.checkURL(value)

	case css.BadURLToken:
		return w.failure("has a url() that it cannot read")

	case css.CommentToken:

		if !isTerminatedComment(data) {
			return w.failure("has a comment that never ends")
		}
	}

	return nil
}

// open records a bracket that a later token must close
func (w *walker) open(opener byte, closer byte, function string) {
	w.blocks = append(w.blocks, block{opener: opener, closer: closer, line: w.line, function: function})
}

// close matches a closing bracket to the innermost open one
func (w *walker) close(closer byte) error {

	// RULE: A closing bracket with nothing open would close whatever wraps the stylesheet
	if len(w.blocks) == 0 {
		return w.failure(fmt.Sprintf("has a %q with nothing to close", closer))
	}

	// RULE: A closing bracket must match the innermost open one
	if innermost := w.blocks[len(w.blocks)-1]; innermost.closer != closer {
		return w.failure(fmt.Sprintf("has a %q where it needs a %q", closer, innermost.closer))
	}

	w.blocks = w.blocks[:len(w.blocks)-1]
	return nil
}

// finish confirms that every block was closed by the end of the stylesheet
func (w *walker) finish() error {

	if len(w.blocks) == 0 {
		return nil
	}

	innermost := w.blocks[len(w.blocks)-1]
	return derp.Validation(fmt.Sprintf("The stylesheet never closes the %q that it opens on line %d.", innermost.opener, innermost.line))
}

// checkImport refuses @import unless files from other websites are allowed
func (w *walker) checkImport() error {

	if w.remoteAllowed {
		return nil
	}

	return w.failure("cannot use @import")
}

// checkURL refuses an address on another website unless files from other websites are allowed
func (w *walker) checkURL(value string) error {

	if w.remoteAllowed {
		return nil
	}

	if !isLocalURL(value) {
		return w.failure("cannot load files from other websites")
	}

	// RULE: A local address must be a file, because some of this website's own pages redirect elsewhere
	if isServerRoute(value) {
		return w.failure("cannot load this website's own pages, only its files")
	}

	return nil
}

// inURLFunction reports whether any open block is a function whose strings are addresses
func (w *walker) inURLFunction() bool {

	// Every open block counts, because a string nested in another function still reaches the outer one
	for _, open := range w.blocks {

		switch open.function {

		case "image", "image-set", "-webkit-image-set", "src", "cross-fade", "-webkit-cross-fade":
			return true
		}
	}

	return false
}

// substitutesAddress reports whether a function would fill in an address that this walk cannot read
func (w *walker) substitutesAddress(name string) bool {

	if w.remoteAllowed {
		return false
	}

	switch name {

	case "var", "env", "attr":
		return w.inURLFunction()
	}

	return false
}

// failure returns a validation error for a rule broken on the current line
func (w *walker) failure(message string) error {
	return derp.Validation(fmt.Sprintf("The stylesheet %s (line %d).", message, w.line))
}
