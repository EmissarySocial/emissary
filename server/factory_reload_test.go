package server

import (
	"embed"
	"sync"
	"testing"

	"github.com/EmissarySocial/emissary/config"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
)

// TestSetupFactory_UnchangedTemplatesKeepServerEmails verifies that reloading a configuration
// with unchanged template locations keeps the server email library loaded.
func TestSetupFactory_UnchangedTemplatesKeepServerEmails(t *testing.T) {

	factory := &SetupFactory{}
	factory.init(&stubStorage{}, embed.FS{})

	t.Cleanup(func() {
		factory.currentWiring().queue.Stop()
	})

	configuration := config.DefaultConfig()
	configuration.Templates = sliceof.Object[mapof.String]{{"adapter": config.FolderAdapterFile, "location": "../_embed/templates"}}

	// Boot with the configuration
	factory.configure(configuration)
	require.NoError(t, factory.Email().RequireModel("user-welcome", "User"))

	// Reload it unchanged, as every write echoes back (BUG-180 emptied the library here)
	factory.configure(configuration)
	require.NoError(t, factory.Email().RequireModel("user-welcome", "User"))
}

// TestSetupFactory_ReadsDuringReload verifies that every template library can be read while a
// genuine template reload writes it.  It proves nothing without -race.
func TestSetupFactory_ReadsDuringReload(t *testing.T) {

	// BUG-180 Defects C and F: these readers once skipped the lock their writers take
	factory := &SetupFactory{}
	factory.init(&stubStorage{}, embed.FS{})

	t.Cleanup(func() {
		factory.currentWiring().queue.Stop()
	})

	// Two spellings of one folder, so that every reload is a genuine one
	configurations := []config.Config{config.DefaultConfig(), config.DefaultConfig()}
	configurations[0].Templates = sliceof.Object[mapof.String]{{"adapter": config.FolderAdapterFile, "location": "../_embed/templates"}}
	configurations[1].Templates = sliceof.Object[mapof.String]{{"adapter": config.FolderAdapterFile, "location": "../_embed/templates/"}}

	factory.configure(configurations[0])

	// Read every library in a loop until the reloads finish
	done := make(chan struct{})
	var waitGroup sync.WaitGroup

	for range 4 {
		waitGroup.Go(func() {
			for {
				select {
				case <-done:
					return
				default:
					_ = factory.Email().RequireModel("user-welcome", "User")
					_ = factory.Template().Names()
					_ = factory.Template().List(nil)
					_ = factory.Theme().List()
					_ = factory.Widget().IsValidWidgetType("html")
					_ = factory.Registration().List()
				}
			}
		})
	}

	// Reload back and forth while the readers run
	for index := range 4 {
		factory.configure(configurations[(index+1)%2])
	}

	close(done)
	waitGroup.Wait()

	require.NoError(t, factory.Email().RequireModel("user-welcome", "User"))
}
