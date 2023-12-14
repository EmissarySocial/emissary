package config

type Option func(*Config)

// WithHTTPPort overrides the HTTP port used by the server
func WithHTTPPort(port int) Option {
	return func(config *Config) {
		config.HTTPPort = port
	}
}
