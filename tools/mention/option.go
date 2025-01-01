package mention

type Option func(*Scanner)

// Hashtags sets the scanner to scan for '#' prefixes only
func Hashtags() Option {
	return func(scanner *Scanner) {
		scanner.prefixes = []rune{'#'}
	}
}

// Mentions sets the scanner to scan for '@' prefixes only
func Mentions() Option {
	return func(scanner *Scanner) {
		scanner.prefixes = []rune{'@'}
	}
}

// IncludePrefix sets the scanner to include the prefix character in the final result
func IncludePrefix() Option {
	return func(scanner *Scanner) {
		scanner.includePrefix = true
	}
}

// WithPrefixes sets the scanner to scan for the provided prefixes
func WithPrefixes(prefixes ...rune) Option {
	return func(scanner *Scanner) {
		scanner.prefixes = prefixes
	}
}

// WithTerminators sets the scanner to scan for the provided terminators
func WithTerminators(terminators ...rune) Option {
	return func(scanner *Scanner) {
		scanner.terminators = terminators
	}
}

// CaseSensitive sets the scanner to return tags in their original case (not lowercased)
func CaseSensitive() Option {
	return func(scanner *Scanner) {
		scanner.caseSensitive = true
	}
}

// CaseInsensitive sets the scanner to return tags in all lowercase
func CaseInsensitive() Option {
	return func(scanner *Scanner) {
		scanner.caseSensitive = true
	}
}
