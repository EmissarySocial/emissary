package asnormalizer

import (
	"slices"
	"strings"
)

// LoadOption is a functional option that the normalizer reads from the options passed to Load.
type LoadOption func(*loadConfig)

// loadConfig holds the options that the normalizer reads from a single Load.
type loadConfig struct {
	history []string
}

// newLoadConfig combines every LoadOption in a Load's options, and ignores options of other types.
func newLoadConfig(options ...any) loadConfig {

	result := loadConfig{}

	for _, option := range options {
		if typed, ok := option.(LoadOption); ok {
			typed(&result)
		}
	}

	return result
}

// WithHistory is a LoadOption that names the documents already being loaded above this Load.
func WithHistory(urls ...string) LoadOption {

	// A variadic argument may be the caller's own slice, so keep a private copy
	urls = slices.Clone(urls)

	return func(config *loadConfig) {
		config.remember(urls...)
	}
}

// remember adds URLs to the history without their fragments, skipping empty values and repeats.
func (config *loadConfig) remember(urls ...string) {

	for _, url := range urls {

		url = withoutFragment(url)

		if url == "" {
			continue
		}

		if config.contains(url) {
			continue
		}

		// Safe to append: every loadConfig builds its own history from nil
		config.history = append(config.history, url)
	}
}

// contains returns TRUE if the history holds this URL, ignoring any fragment.
func (config loadConfig) contains(url string) bool {
	return slices.Contains(config.history, withoutFragment(url))
}

// isTooDeep returns TRUE if the history is long enough that no further documents may be loaded.
func (config loadConfig) isTooDeep() bool {
	return len(config.history) >= maxDepth
}

// withoutFragment returns a URL with its "#fragment" removed.
func withoutFragment(url string) string {
	result, _, _ := strings.Cut(url, "#")
	return result
}
