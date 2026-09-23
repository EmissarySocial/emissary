package server

import (
	"embed"
	"testing"

	"github.com/EmissarySocial/emissary/config"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
)

// TestSetupFactory_UnchangedTemplatesKeepServerEmails is the BUG-180 outage, through the real
// reload path: every configuration write echoes back to every node, and a reload whose template
// locations had not changed emptied the server email library, so registration failed until restart.
func TestSetupFactory_UnchangedTemplatesKeepServerEmails(t *testing.T) {

	factory := &SetupFactory{}
	factory.init(&stubStorage{}, embed.FS{})

	t.Cleanup(func() {
		factory.currentWiring().queue.Stop()
	})

	configuration := config.DefaultConfig()
	configuration.Templates = sliceof.Object[mapof.String]{{"adapter": config.FolderAdapterFile, "location": "../_embed/templates"}}

	// Boot, then the echo of any configuration write
	factory.configure(configuration)
	require.NoError(t, factory.Email().RequireModel("user-welcome", "User"))

	factory.configure(configuration)
	require.NoError(t, factory.Email().RequireModel("user-welcome", "User"))
}
