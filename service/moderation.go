package service

import "github.com/EmissarySocial/emissary/service/moderation"

// Moderation wraps a pluggable moderation backend selected from domain config.
type Moderation struct {
	moderation.Moderation
}

func NewModeration() Moderation {
	return Moderation{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates the moderation implementation based on the current domain config.
func (m *Moderation) Refresh(factory *Factory) {
	m.Moderation = moderation.NewModeration(factory.Config().Moderation)
}
