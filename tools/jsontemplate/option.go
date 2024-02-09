package jsontemplate

type Option func(*Template)

// WithStrictMode sets the strict mode option for the template, using the standard Go unmarshaller for JSON
func WithStrictMode() Option {
	return func(t *Template) {
		t.strictMode = true
	}
}
