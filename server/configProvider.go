package server

import (
	"github.com/EmissarySocial/emissary/config"
)

// ConfigProvider is the minimal factory surface needed by code that must read the LIVE server
// configuration instead of a snapshot taken when it was built.  Both Factory and SetupFactory
// satisfy it.
type ConfigProvider interface {

	// Config returns an independent copy of the current server configuration
	Config() config.Config
}
