package ascrawler

type LoadOption func(*loadConfig)

// WithHistory adds a URI into the load history. This is used to prevent
// infinite loops when loading documents.
func WithHistory(uris ...string) LoadOption {
	return func(config *loadConfig) {
		config.history = append(config.history, uris...)
	}
}
