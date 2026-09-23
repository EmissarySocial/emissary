package service

import (
	"os"
	"testing"
	"testing/fstest"

	"github.com/EmissarySocial/emissary/config"
	emissarytemplates "github.com/EmissarySocial/emissary/tools/templates"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// refreshTestTemplate returns a Template service loaded from the "first" of two embedded
// template folders, each holding the shipped user-welcome email
func refreshTestTemplate(t *testing.T) *Template {

	t.Helper()

	definition, err := os.ReadFile("../_embed/templates/email-user-welcome/email.hjson")
	require.NoError(t, err)

	body, err := os.ReadFile("../_embed/templates/email-user-welcome/body.html")
	require.NoError(t, err)

	embedded := fstest.MapFS{
		"_embed/first/email-user-welcome/email.hjson":  {Data: definition},
		"_embed/first/email-user-welcome/body.html":    {Data: body},
		"_embed/second/email-user-welcome/email.hjson": {Data: definition},
		"_embed/second/email-user-welcome/body.html":   {Data: body},
	}

	funcMap := emissarytemplates.FuncMap(nullIconProvider{})
	filesystemService := NewFilesystem(embedded)
	emailService := NewServerEmail(filesystemService, funcMap, nil)

	return NewTemplate(filesystemService, &Registration{}, &emailService, &Theme{}, &Widget{}, funcMap, refreshTestLocation("first"))
}

// refreshTestLocation returns the template configuration for one embedded test folder
func refreshTestLocation(folder string) []mapof.String {
	return []mapof.String{{"adapter": config.FolderAdapterEmbed, "location": folder}}
}

// TestTemplateRefresh_UnchangedLocationsKeepWatcher is BUG-180 Defect B: a configuration reload
// whose template locations had not changed stopped the file watcher and started nothing in its
// place, so edits on disk were ignored until a restart.
func TestTemplateRefresh_UnchangedLocationsKeepWatcher(t *testing.T) {

	templateService := refreshTestTemplate(t)
	watcher := templateService.refresh

	templateService.Refresh(refreshTestLocation("first"))

	select {
	case <-watcher:
		t.Fatal("a reload with unchanged locations stopped the template watcher")
	default:
	}
}

// TestTemplateRefresh_ChangedLocationsReplaceWatcher verifies that a genuine reload still stops
// the old watcher, which is watching folders that are no longer configured
func TestTemplateRefresh_ChangedLocationsReplaceWatcher(t *testing.T) {

	templateService := refreshTestTemplate(t)
	watcher := templateService.refresh

	templateService.Refresh(refreshTestLocation("second"))

	select {
	case <-watcher:
	default:
		t.Fatal("a reload with new locations left the old watcher running")
	}

	require.NotEqual(t, watcher, templateService.refresh, "the new watcher needs a channel of its own")
}
