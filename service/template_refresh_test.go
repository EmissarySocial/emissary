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
// template folders, each holding the shipped user-welcome email, with hot reload on
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

	result := NewTemplate(filesystemService, &Registration{}, &emailService, &Theme{}, &Widget{}, funcMap, refreshTestLocation("first"))
	result.Refresh(refreshTestLocation("first"), true)

	return result
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

	templateService.Refresh(refreshTestLocation("first"), true)

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

	templateService.Refresh(refreshTestLocation("second"), true)

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
		templateService.Refresh(location(index+1), true)
	}

	require.NoError(t, emailService.RequireModel("email-0", "User"))
}

// TestTemplateRefresh_HotReloadOffIgnoresChanges verifies that templates on disk are not
// watched unless hot reload is on, which is the default
func TestTemplateRefresh_HotReloadOffIgnoresChanges(t *testing.T) {

	templateService, emailService, folder := hotReloadTestTemplate(t)
	templateService.Refresh(hotReloadTestLocation(folder), false)

	writeHotReloadEmail(t, folder, "email-added")
	requireNoHotReload(t, emailService, "email-added")
}

// TestTemplateRefresh_HotReloadOnWatchesChanges verifies that turning hot reload on, with the
// locations unchanged, starts a watcher that reloads templates when a file changes
func TestTemplateRefresh_HotReloadOnWatchesChanges(t *testing.T) {

	templateService, emailService, folder := hotReloadTestTemplate(t)
	watcher := templateService.refresh

	templateService.Refresh(hotReloadTestLocation(folder), true)
	require.NotEqual(t, watcher, templateService.refresh, "turning hot reload on must start a watcher")

	// Retry the write, because the watcher starts asynchronously and misses changes made before it does
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {

		writeHotReloadEmail(t, folder, "email-added")

		if emailService.RequireModel("email-added", "User") == nil {
			return
		}
	}

	t.Fatal("hot reload never loaded the new template")
}

// TestTemplateRefresh_HotReloadOffStopsWatcher verifies that turning hot reload off, with the
// locations unchanged, stops the running watcher
func TestTemplateRefresh_HotReloadOffStopsWatcher(t *testing.T) {

	templateService, emailService, folder := hotReloadTestTemplate(t)
	templateService.Refresh(hotReloadTestLocation(folder), true)
	watcher := templateService.refresh

	templateService.Refresh(hotReloadTestLocation(folder), false)

	select {
	case <-watcher:
	default:
		t.Fatal("turning hot reload off left the watcher running")
	}

	// Give the stopped watcher's goroutines time to exit before changing the folder
	time.Sleep(100 * time.Millisecond)

	writeHotReloadEmail(t, folder, "email-added")
	requireNoHotReload(t, emailService, "email-added")
}

// hotReloadTestTemplate returns a Template service loaded from a temporary folder on disk,
// with hot reload off, along with its email service and the folder
func hotReloadTestTemplate(t *testing.T) (*Template, *ServerEmail, string) {

	t.Helper()

	folder := t.TempDir()
	writeHotReloadEmail(t, folder, "email-initial")

	funcMap := emissarytemplates.FuncMap(nullIconProvider{})
	filesystemService := NewFilesystem(nil)
	emailService := NewServerEmail(filesystemService, funcMap, nil)
	templateService := NewTemplate(filesystemService, &Registration{}, &emailService, &Theme{}, &Widget{}, funcMap, hotReloadTestLocation(folder))

	// Stop the current watcher when the test ends, so that it does not outlive the folder
	t.Cleanup(func() { close(templateService.refresh) })

	require.NoError(t, emailService.RequireModel("email-initial", "User"))
	return templateService, &emailService, folder
}

// hotReloadTestLocation returns the template configuration for one folder on disk
func hotReloadTestLocation(folder string) []mapof.String {
	return []mapof.String{{"adapter": config.FolderAdapterFile, "location": folder}}
}

// writeHotReloadEmail writes a copy of the shipped user-welcome email into the folder, as emailID
func writeHotReloadEmail(t *testing.T, folder string, emailID string) {

	t.Helper()

	definition, err := os.ReadFile("../_embed/templates/email-user-welcome/email.hjson")
	require.NoError(t, err)

	body, err := os.ReadFile("../_embed/templates/email-user-welcome/body.html")
	require.NoError(t, err)

	directory := filepath.Join(folder, emailID)
	require.NoError(t, os.MkdirAll(directory, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "email.hjson"), []byte(strings.Replace(string(definition), "user-welcome", emailID, 1)), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "body.html"), body, 0o600))
}

// requireNoHotReload fails if the email template is loaded within half a second
func requireNoHotReload(t *testing.T, emailService *ServerEmail, emailID string) {

	t.Helper()

	time.Sleep(500 * time.Millisecond)
	require.Error(t, emailService.RequireModel(emailID, "User"), "the template was reloaded without hot reload")
}
