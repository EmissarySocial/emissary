package parse

// softTerminators is a list of characters that might end a token, but only if followed by a whitespace character.
var softTerminators = []rune{'.', ':', '+', '/', '\\', '|'}

// baseTerminators always end a token, whatever kind of token it is.
var baseTerminators = []rune{' ', '\t', '\n', '\r', ',', '!', '?', ';', '%', '(', ')', '[', ']', '{', '}', '<', '>', '`', '"', '\''}

// hashtagTerminators additionally break on the punctuation that no other fediverse server allows
// inside a tag.  See bugs/BUG-69 in emissary-specs.
//
// The '-' is the one that matters in practice: every other implementation reads "#well-being" as
// "#well", so a tag Emissary publishes with the hyphen inside it is a tag nobody else indexes.
// '#' and '@' are here because they are prefix characters -- neither can appear inside a token.
//
// NOTE: this stays a DENY-list.  FEP-eb48 rule 2 specifies an allow-list of "A-Z, a-z, 0-9", which
// read literally excludes "#日本語", "#Привет", and "#café" -- all of which work everywhere else.
// '_' is deliberately absent: underscores are valid inside hashtags.
var hashtagTerminators = concatRunes(baseTerminators, '#', '@', '-', '&', '^', '~', '=', '*', '$')

// mentionTerminators break on the same decorative punctuation as hashtags, but NOT on '@' or '-',
// because a WebFinger handle needs both: "@user@my-host.social" is one token, not three.
var mentionTerminators = concatRunes(baseTerminators, '#', '&', '^', '~', '=', '*', '$')

// whitespace is a list of characters that are considered "whitespace" for the purposes of parsing.
var whitespace = []rune{' ', '\t', '\n', '\r'}

// concatRunes returns a new slice, so that the base list is never aliased or appended to in place.
func concatRunes(base []rune, extra ...rune) []rune {
	result := make([]rune, 0, len(base)+len(extra))
	result = append(result, base...)
	result = append(result, extra...)
	return result
}
