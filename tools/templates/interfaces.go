package templates

type Replacer interface {
	Replace(string) string
}
