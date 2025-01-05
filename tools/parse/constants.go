package parse

// softTerminators is a list of characters that might end a token, but only if followed by a whitespace character.
var softTerminators = []rune{'.', ':', '+', '/', '\\', '|'}

// hardTerminators is a list of characters that always end a token, regardless of what comes next.
var hardTerminators = []rune{' ', '\t', '\n', '\r', ',', '!', '?', ';', '%', '(', ')', '[', ']', '{', '}', '<', '>', '`', '"', '\''}

// whitespace is a list of characters that are considered "whitespace" for the purposes of parsing.
var whitespace = []rune{' ', '\t', '\n', '\r'}
