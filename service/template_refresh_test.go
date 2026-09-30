package service

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"

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

// TestTemplateRefresh_UnchangedLocationsKeepWatcher verifies that a reload whose locations
// have not changed leaves the file watcher running
func TestTemplateRefresh_UnchangedLocationsKeepWatcher(t *testing.T) {

	// BUG-180 Defect B: this reload stopped the watcher and started nothing in its place,
	// so edits on disk were ignored until a restart
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

// TestTemplateRefresh_WatcherAndReloadDoNotOverlap verifies that a file change and a
// configuration reload never run loadTemplates at the same time
func TestTemplateRefresh_WatcherAndReloadDoNotOverlap(t *testing.T) {

	// BUG-180 Defect D: both loads wrote the shared prep area at once.  This test
	// depends on timing, so it proves nothing without -race.
	definition, err := os.ReadFile("../_embed/templates/email-user-welcome/email.hjson")
	require.NoError(t, err)

	body, err := os.ReadFile("../_embed/templates/email-user-welcome/body.html")
	require.NoError(t, err)

	// Two watched folders on disk, so that a file change reaches the watcher's own reload.  Each
	// holds 100 emails, so that a load lasts long enough for the other one to start during it.
	folders := []string{t.TempDir(), t.TempDir()}

	for _, folder := range folders {
		for index := range 100 {
			emailID := "email-" + strconv.Itoa(index)
			directory := filepath.Join(folder, emailID)
			require.NoError(t, os.Mkdir(directory, 0o700))
			require.NoError(t, os.WriteFile(filepath.Join(directory, "email.hjson"), []byte(strings.Replace(string(definition), "user-welcome", emailID, 1)), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(directory, "body.html"), body, 0o600))
		}
	}

	location := func(index int) []mapof.String {
		return []mapof.String{{"adapter": config.FolderAdapterFile, "location": folders[index%2]}}
	}

	funcMap := emissarytemplates.FuncMap(nullIconProvider{})
	filesystemService := NewFilesystem(nil)
	emailService := NewServerEmail(filesystemService, funcMap, nil)
	templateService := NewTemplate(filesystemService, &Registration{}, &emailService, &Theme{}, &Widget{}, funcMap, location(0))

	// Each round lets the new watcher start, creates a file so that it begins a reload, and then
	// reloads to the other folder while that one is still running
	for index := range 10 {
		time.Sleep(100 * time.Millisecond)
		require.NoError(t, os.WriteFile(filepath.Join(folders[index%2], "email-0", "change-"+strconv.Itoa(index)), nil, 0o600))
		time.Sleep(time.Duration(index) * time.Millisecond)
		templateService.Refresh(location(index + 1))
	}

	require.NoError(t, emailService.RequireModel("email-0", "User"))
}
