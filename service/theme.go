package service

import (
	"html/template"
	"io/fs"
	"sort"
	"sync"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/slice"
	"github.com/hjson/hjson-go/v4"
	"github.com/rs/zerolog/log"
)

// Theme service manages the global site theme that is stored in a particular path of the
// filesystem.
type Theme struct {
	templateService *Template
	contentService  *Content
	funcMap         template.FuncMap
	themes          mapof.Object[model.Theme]

	mutex   sync.RWMutex
	changed chan bool
	closed  chan bool
}

// NewTheme returns a fully initialized Theme service.
func NewTheme(templateService *Template, contentService *Content, funcMap template.FuncMap) Theme {

	return Theme{
		templateService: templateService,
		contentService:  contentService,
		funcMap:         funcMap,
		themes:          mapof.NewObject[model.Theme](),
		mutex:           sync.RWMutex{},
		changed:         make(chan bool),
		closed:          make(chan bool),
	}
}

/******************************************
 * Data Access Methods
 ******************************************/

// List returns an iterator over the Theme records that match the provided criteria
func (service *Theme) List() []model.Theme {

	// Generate a slice containing all themes
	result := make([]model.Theme, 0, len(service.themes))

	// Lock the data structure
	service.mutex.RLock()
	defer service.mutex.RUnlock()

	for _, theme := range service.themes {
		if theme.IsVisible {
			result = append(result, theme)
		}
	}

	return result
}

// ListSorted returns every Theme, in display order
func (service *Theme) ListSorted() []model.Theme {
	result := service.List()

	sort.Slice(result, func(i, j int) bool {
		return model.SortThemes(result[i], result[j])
	})

	return result
}

// ListActive returns every Theme that is not a placeholder, in display order
func (service *Theme) ListActive() []model.Theme {
	return slice.Filter(service.ListSorted(), func(theme model.Theme) bool {
		return !theme.IsPlaceholder()
	})
}

// GetTheme returns the named Theme, falling back to the default Theme if it does not exist
func (service *Theme) GetTheme(themeID string) model.Theme {

	service.mutex.RLock()
	defer service.mutex.RUnlock()

	// Try to return the requested theme.
	// This should usually happen
	if theme, ok := service.themes[themeID]; ok {
		return theme
	}

	// If the requested theme doesn't exist, then return the default theme.
	// This should rarely happen
	if theme, ok := service.themes["default"]; ok {
		return theme
	}

	// If the default theme doesn't exist, then return a blank theme.
	// This should never happen, and it'll probably break when you try to run it.
	return model.NewTheme("default", service.funcMap)
}

/******************************************
 * Loading Themes
 ******************************************/

// Add parses a theme definition and registers it under the provided themeID
func (service *Theme) Add(themeID string, filesystem fs.FS, definition []byte) error {

	const location = "service.Theme.loadModel"

	log.Debug().Msg("Theme Service: adding theme: " + themeID)

	theme := model.NewTheme(themeID, service.funcMap)

	// Try to parse the JSON in the buffer into a Theme object
	if err := hjson.Unmarshal(definition, &theme); err != nil {
		return derp.Wrap(err, location, "Parsing theme.json file", filesystem)
	}

	// Every format name in the schema must resolve in the format registry; unrecognized
	// names are silently skipped at validation time, so reject them at load time instead.
	if err := theme.Schema.ValidateFormats(); err != nil {
		return derp.Wrap(err, location, "Theme schema uses an unrecognized format name", themeID)
	}

	// Load HTML templates into the theme
	if err := loadHTMLTemplateFromFilesystem(filesystem, theme.HTMLTemplate, service.funcMap); err != nil {
		return derp.Wrap(err, "service.theme.loadFromFilesystem", "Loading Template", themeID)
	}

	// Load all Bundles from the filesystem
	if err := populateBundles(theme.Bundles, filesystem); err != nil {
		return derp.Wrap(err, "service.template.loadFromFilesystem", "Loading Bundles", themeID)
	}

	// Keep a pointer to the filesystem resources (if present)
	if resources, err := fs.Sub(filesystem, "resources"); err == nil {
		theme.Resources = resources
	}

	if content, err := fs.Sub(filesystem, "content"); err == nil {
		service.setStartupContent(&theme, content)
	}

	// Add the theme into the theme library
	service.set(theme)
	return nil
}

// set writes a Theme into the in-memory map, guarding against concurrent readers
func (service *Theme) set(theme model.Theme) {

	service.mutex.Lock()
	defer service.mutex.Unlock()

	service.themes[theme.ThemeID] = theme
}

// calculateAllInheritance re-applies inheritance to every registered Theme
func (service *Theme) calculateAllInheritance() {

	for _, theme := range service.themes {
		service.calculateInheritance(theme)
	}
}

// calculateInheritance fills in a Theme's empty values from each of the Themes it extends
func (service *Theme) calculateInheritance(theme model.Theme) model.Theme {

	if len(theme.Extends) == 0 {
		return theme
	}

	for _, parentName := range theme.Extends {
		if parent, ok := service.themes[parentName]; ok {
			parent = service.calculateInheritance(parent)
			theme.Inherit(&parent)
		}
	}

	service.set(theme)
	return theme
}

// setStartupContent loads the sample content that a new Domain is seeded with from this Theme
func (service *Theme) setStartupContent(theme *model.Theme, filesystem fs.FS) {

	// Try to read files in the "content" directory
	entries, err := fs.ReadDir(filesystem, ".")

	if err != nil {
		return
	}

	// For each file in the directory
	for _, entry := range entries {

		// Search for a StartupStream with a matching filename
		for index := range theme.StartupStreams {
			filename, extension := list.Split(entry.Name(), '.') // nolint:scopeguard (readability)
			if theme.StartupStreams[index].GetString("token") == filename {

				// If there is a match, then load the file into the StartupStream
				if raw, err := fs.ReadFile(filesystem, entry.Name()); err == nil {
					theme.StartupStreams[index]["content"] = service.contentService.NewByExtension(extension, string(raw))
					break
				}
			}
		}
	}
}
