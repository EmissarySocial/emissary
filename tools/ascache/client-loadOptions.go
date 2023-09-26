package ascache

type LoadConfig struct {
	forceReload bool
}

// isCacheAllowed returns TRUE if the cache is allowed to be used for this request.
func (config LoadConfig) isCacheAllowed() bool {
	return !config.forceReload
}

type LoadOption func(*LoadConfig)

func NewLoadConfig(options ...any) LoadConfig {
	result := LoadConfig{
		forceReload: false,
	}

	result.With(options...)
	return result
}

func (config *LoadConfig) With(options ...any) {
	for _, option := range options {
		if typed, ok := option.(LoadOption); ok {
			typed(config)
		}
	}
}

func WithForceReload() LoadOption {
	return func(config *LoadConfig) {
		config.forceReload = true
	}
}

func WithoutForceReload() LoadOption {
	return func(config *LoadConfig) {
		config.forceReload = false
	}
}
