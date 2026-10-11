package tests

import (
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/templates"
	"github.com/benpate/form/widget"
	"github.com/benpate/rosetta/mapof"
	"github.com/davidscottmills/goeditorjs"
	"github.com/stretchr/testify/require"
)

// templateGroup is one deployment's worth of template locations, loaded together the way
// a server's configuration loads them: Emissary's own templates first, then a package.
type templateGroup struct {
	Name      string   // Name used for golden files and subtests
	Locations []string // Directories, relative to this package, in load order
}

// templateGroups lists every deployment this suite pins.  Each package directory is a
// sibling checkout of the emissary repository.
var templateGroups = []templateGroup{
	{Name: "emissary", Locations: []string{emissaryTemplates}},
	{Name: "atlas", Locations: []string{emissaryTemplates, "../../atlas"}},
	{Name: "bandwagon", Locations: []string{emissaryTemplates, "../../bandwagon"}},
	{Name: "qwertylicious", Locations: []string{emissaryTemplates, "../../qwertylicious"}},
	{Name: "qwertylicious-v2", Locations: []string{emissaryTemplates, "../../qwertylicious-v2"}},
}

// emissaryTemplates is the folder that holds every template that ships in the binary
const emissaryTemplates = "../_embed/templates"

// loadedGroup holds the production services after they have loaded one templateGroup
type loadedGroup struct {
	Templates     *service.Template
	Themes        *service.Theme
	Widgets       *service.Widget
	Registrations *service.Registration
}

var (
	loadedGroups     = map[string]*loadedGroup{}
	loadedGroupsLock sync.Mutex
)

// loadGroup loads a templateGroup through the production services, wired as the server's
// factory wires them, and caches it.  A group that is not checked out is skipped.
func loadGroup(t testing.TB, group templateGroup) *loadedGroup {

	t.Helper()

	// RULE: A package that is not checked out beside emissary cannot be pinned here
	if missing := group.missingLocation(); missing != "" {
		t.Skipf("template location %q is not checked out", missing)
	}

	loadedGroupsLock.Lock()
	defer loadedGroupsLock.Unlock()

	if result, exists := loadedGroups[group.Name]; exists {
		return result
	}

	// Register every form widget, as the server does at startup
	widget.UseAll()

	// Wire the services in the same order as server/factory_core.go
	funcMap := templates.FuncMap(service.NewIcons())
	filesystemService := service.NewFilesystem(nil)
	registrationService := service.NewRegistration(funcMap)
	contentService := service.NewContent(editorJS())
	widgetService := service.NewWidget(funcMap)
	emailService := service.NewServerEmail(filesystemService, funcMap, nil)

	result := &loadedGroup{
		Templates:     &service.Template{},
		Widgets:       &widgetService,
		Registrations: &registrationService,
	}

	themeService := service.NewTheme(result.Templates, &contentService, funcMap)
	result.Themes = &themeService

	*result.Templates = *service.NewTemplate(filesystemService, &registrationService, &emailService, &themeService, &widgetService, funcMap, nil) // nolint:govet (copies the service before its first use, as the factory does)

	// Load every location, in order, through the production loader
	locations := make([]mapof.String, 0, len(group.Locations))
	for _, location := range group.Locations {
		absolute, err := filepath.Abs(location)
		require.NoError(t, err)
		locations = append(locations, mapof.String{"adapter": config.FolderAdapterFile, "location": absolute})
	}

	result.Templates.Refresh(locations, false)

	// A silent zero would let every test below pass without pinning anything
	require.NotEmpty(t, result.Templates.Names(), "no templates loaded for %s", group.Name)

	loadedGroups[group.Name] = result
	return result
}

// missingLocation returns the first location that is not checked out, or ""
func (group templateGroup) missingLocation() string {
	for _, location := range group.Locations {
		if _, err := os.Stat(location); err != nil {
			return location
		}
	}
	return ""
}

// templateIDs returns every loaded template, sorted so that subtests run in a stable order
func (group *loadedGroup) templateIDs() []string {
	result := group.Templates.Names()
	sort.Strings(result)
	return result
}

// themes returns every loaded theme, sorted by ID
func (group *loadedGroup) themes() []model.Theme {
	result := group.Themes.List()
	sort.Slice(result, func(i, j int) bool { return result[i].ThemeID < result[j].ThemeID })
	return result
}

// editorJS returns the same Editor.js engine that server/factory_core.go builds
func editorJS() *goeditorjs.HTMLEngine {

	result := goeditorjs.NewHTMLEngine()

	result.RegisterBlockHandlers(
		&goeditorjs.HeaderHandler{},
		&goeditorjs.ParagraphHandler{},
		&goeditorjs.ListHandler{},
		&goeditorjs.ImageHandler{},
		&goeditorjs.RawHTMLHandler{},
	)

	return result
}
