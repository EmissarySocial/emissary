package ascontextmaker

// LoadConfig contains optional settings for the Load() method
type LoadConfig struct {
	history []string // history tracks URI's that have already been loaded
}

// LoadOption is a function that modifies the default behavior of the LoadConfig
type LoadOption func(*LoadConfig)

// NewLoadConfig creates a new LoadConfig object with corresponding options
func NewLoadConfig(options ...any) LoadConfig {
	result := LoadConfig{
		history: make([]string, 0),
	}

	result.With(options...)
	return result
}

// With applies options to the LoadConfig IF they are ascontextmaker.LoadOption
func (config *LoadConfig) With(options ...any) {
	for _, option := range options {
		if typed, ok := option.(LoadOption); ok {
			typed(config)
		}
	}
}

// WithHistory adds a URI into the load history. This is used to prevent
// infinite loops when loading documents.
func WithHistory(uris ...string) LoadOption {
	return func(config *LoadConfig) {
		config.history = append(config.history, uris...)
	}
}
