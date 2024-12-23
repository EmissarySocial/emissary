package mention

// Parse is a convenienct function that parses values using the default settings
func Parse(prefix rune, original string) ([]string, string) {
	return New(WithPrefixes(prefix)).Parse(original)
}

// ParseTagsOnly is a convenienct function that parses values using the default settings
func ParseTagsOnly(prefix rune, original string) []string {
	return New(WithPrefixes(prefix)).ParseTagsOnly(original)
}
