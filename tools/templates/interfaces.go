package templates

// Replacer transforms a string, and is implemented by anything that rewrites template output
type Replacer interface {
	Replace(string) string
}
