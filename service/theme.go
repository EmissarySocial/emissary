package service

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"sync"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/slice"
	"golang.org/x/exp/slices"
)

// Theme service manages the global site theme that is stored in a particular path of the
// filesystem.
type Theme struct {
	templateService *Template
	funcMap         template.FuncMap
	themes          mapof.Object[model.Theme]

	mutex   sync.RWMutex
	changed chan bool
	closed  chan bool
}

// NewTheme returns a fully initialized Theme service.
func NewTheme(templateService *Template, funcMap template.FuncMap) *Theme {

	service := Theme{
		templateService: templateService,
		funcMap:         funcMap,
		themes:          mapof.NewObject[model.Theme](),
		mutex:           sync.RWMutex{},
		changed:         make(chan bool),
		closed:          make(chan bool),
	}

	return &service
}

/******************************************
 * Data Access Methods
 ******************************************/

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

func (service *Theme) ListSorted() []model.Theme {
	result := service.List()
	slices.SortFunc(result, model.SortThemes)

	return result
}

func (service *Theme) ListActive() []model.Theme {
	return slice.Filter(service.ListSorted(), func(theme model.Theme) bool {
		return !theme.IsPlaceholder()
	})
}

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

func (service *Theme) Add(themeID string, filesystem fs.FS, definition []byte) error {

	const location = "service.Theme.loadModel"

	theme := model.NewTheme(themeID, service.funcMap)

	// Try to parse the JSON in the buffer into a Theme object
	if err := json.Unmarshal(definition, &theme); err != nil {
		return derp.Wrap(err, location, "Unable to parse theme.json file", filesystem)
	}

	// Load HTML templates into the theme
	if err := loadHTMLTemplateFromFilesystem(filesystem, theme.HTMLTemplate, service.funcMap); err != nil {
		return derp.Wrap(err, "service.theme.loadFromFilesystem", "Error loading Template", themeID)
	}

	// Load all Bundles from the filesystem
	if err := populateBundles(theme.Bundles, filesystem); err != nil {
		return derp.Wrap(err, "service.template.loadFromFilesystem", "Error loading Bundles", themeID)
	}

	// Keep a pointer to the filesystem resources (if present)
	if resources, err := fs.Sub(filesystem, "resources"); err == nil {
		theme.Resources = resources
	}

	fmt.Println("... adding theme: " + theme.ThemeID)

	// Add the theme into the theme library
	service.set(theme)
	return nil
}

func (service *Theme) set(theme model.Theme) {

	service.mutex.Lock()
	defer service.mutex.Unlock()

	service.themes[theme.ThemeID] = theme
}

func (service *Theme) calculateAllInheritance() {

	for _, theme := range service.themes {
		service.calculateInheritance(theme)
	}
}

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
