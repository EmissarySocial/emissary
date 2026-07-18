package asrules

// loadConfig collects the per-call options for Load.
type loadConfig struct {
	reveal bool
}

// LoadOption is a functional option that can be passed to the Load() method.
type LoadOption func(*loadConfig)

// newLoadConfig builds a loadConfig from the options passed to Load, ignoring options that belong
// to other clients in the stack.
func newLoadConfig(options ...any) loadConfig {

	result := loadConfig{}

	for _, option := range options {
		if typed, ok := option.(LoadOption); ok {
			typed(&result)
		}
	}

	return result
}

// WithReveal loads a document even when the viewer's rules would hide it. The verdict is still
// computed and stamped into the document's Metadata -- this option only lifts the refusal. It is
// the click-to-reveal override (D2): the UX passes it when the user asks to see hidden content
// despite a block or mute.
func WithReveal(reveal bool) LoadOption {
	return func(config *loadConfig) {
		config.reveal = reveal
	}
}
