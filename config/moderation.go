package config

import (
	"time"

	"github.com/benpate/derp"
)

// ModerationProviderCoop is the provider name for the Coop moderation backend.
const ModerationProviderCoop = "coop"

// Moderation contains per-domain configuration for external moderation backends.
type Moderation struct {
	Provider string `json:"provider" bson:"provider"` // Name of the moderation backend to use (e.g. "coop"). Empty means no backend configured.
	URL      string `json:"url"      bson:"url"`      // Base URL of the moderation backend (e.g. "http://coop:3000"). Per-domain, not server-level.
	Coop     Coop   `json:"coop"     bson:"coop"`     // Configuration for the Coop moderation backend
}

// NewModeration returns a fully initialized (empty) Moderation config.
func NewModeration() Moderation {
	return Moderation{}
}

// IsNil returns TRUE if no moderation backend is configured.
func (m Moderation) IsNil() bool {
	return m.Provider == ""
}

// TestConnection verifies the configured moderation backend is reachable.
// An unconfigured (empty) block is a no-op success.
func (m Moderation) TestConnection(timeout time.Duration) error {

	const location = "config.Moderation.TestConnection"

	if m.IsNil() {
		return nil
	}

	if m.URL == "" {
		return derp.Validation("Moderation provider is selected but no URL is configured. Set the Moderation Backend URL in the domain's Moderation settings.")
	}

	switch m.Provider {

	case ModerationProviderCoop:
		return m.Coop.TestConnection(m.URL, timeout)

	default:
		return derp.Validation("Unknown moderation provider: "+m.Provider, m.Provider)
	}
}
