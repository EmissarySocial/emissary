package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"sync"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/slice"
)

// Theme service manages the global site theme that is stored in a particular path of the
// filesystem.
type Theme struct {
	filesystemService Filesystem
	funcMap           template.FuncMap
	locations         []config.Folder
	themes            mapof.Object[model.Theme]

	mutex   sync.RWMutex
	changed chan bool
	closed  chan bool
}

// NewTheme returns a fully initialized Theme service.
func NewTheme(filesystemService Filesystem, funcMap template.FuncMap, locations []config.Folder) *Theme {

	service := Theme{
		filesystemService: filesystemService,
		funcMap:           funcMap,
		locations:         make([]config.Folder, 0),
		themes:            mapof.NewObject[model.Theme](),
		mutex:             sync.RWMutex{},
		changed:           make(chan bool),
		closed:            make(chan bool),
	}

	service.Refresh(locations)

	return &service
}

/******************************************
 * Lifecycle Methods
 ******************************************/

func (service *Theme) Refresh(locations []config.Folder) {

	// If nothing has changed since the last time we refreshed, then we're done
	if slice.Equal(locations, service.locations) {
		return
	}

	fmt.Println("Refreshing theme")
	service.locations = locations

	// Try to load themes from the filesystems
	if err := service.loadThemes(); err != nil {
		derp.Report(err)
	}

	go service.Watch()
}

/******************************************
 * Data Access Methods
 ******************************************/

func (service *Theme) List() []model.Theme {

	service.mutex.RLock()
	defer service.mutex.RUnlock()

	// Generate a slice containing all themes
	result := make([]model.Theme, 0, len(service.themes))

	for _, theme := range service.themes {
		result = append(result, theme)
	}

	return result
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
 * Real-Time Updates
 ******************************************/

// watch must be run as a goroutine, and constantly monitors the
// "Updates" channel for news that a template has been updated.
func (service *Theme) Watch() {

	// Close the channel, which will stopp any existing watchers
	close(service.closed)

	// Create a new "closed" channel to close future watchers
	service.closed = make(chan bool)

	// Start new watchers.
	for _, folder := range service.locations {

		// for _, filename := range service.fileNames() {
		service.filesystemService.Watch(folder, service.changed, service.closed) // Fail silently because many locations may not define all themes
		// }
	}

	// All Watchers Started.  Now Listen for Changes
	for {

		select {

		case <-service.changed:
			service.loadThemes()

		case <-service.closed:
			return
		}
	}
}

// loadThemes retrieves the template from the disk and parses it into
func (service *Theme) loadThemes() error {

	result := mapof.NewObject[model.Theme]()

	// For each configured location...
	for _, location := range service.locations {

		fmt.Println("Searching for themes in: " + location.Location)

		// Get a valid filesystem adapter
		filesystem, err := service.filesystemService.GetFS(location)

		if err != nil {
			derp.Report(derp.Wrap(err, "service.Theme.loadThemes", "Unable to get filesystem adapter", location))
			continue // If there's an error, it means that this location just doesn't define this part of the theme.  It's OK
		}

		themes, err := fs.ReadDir(filesystem, ".")

		if err != nil {
			return derp.Wrap(err, "service.loadHTMLTemplateFromFilesystem", "Unable to list themes in directory", filesystem)
		}

		// Check each of the standard filenames
		for _, themeDirectory := range themes {

			// Skip non-directories
			if !themeDirectory.IsDir() {
				continue
			}

			themeID := themeDirectory.Name()
			subdirectory, err := fs.Sub(filesystem, themeID)

			if err != nil {
				derp.Report(derp.NewInternalError("service.theme.loadFromFilesystem", "Error loading subdirectory", location, themeID))
				continue
			}

			// Read the theme.json file
			theme, err := service.loadModel(themeID, subdirectory)

			if err != nil {
				derp.Report(derp.Wrap(err, "service.theme.loadFromFilesystem", "Error loading theme.json", location, themeID))
				continue
			}

			// Load HTML templates into the theme
			if err := loadHTMLTemplateFromFilesystem(subdirectory, theme.HTMLTemplate, service.funcMap); err != nil {
				derp.Report(derp.Wrap(err, "service.theme.loadFromFilesystem", "Error loading Template", location, themeID))
				continue
			}

			// Load all Bundles from the filesystem
			if err := populateBundles(theme.Bundles, subdirectory); err != nil {
				derp.Report(derp.Wrap(err, "service.template.loadFromFilesystem", "Error loading Bundles", location))
				continue
			}

			result[theme.ThemeID] = theme
			fmt.Println(". Success.")
		}
	}

	// Apply all themes at once to minimize lock time.
	service.mutex.Lock()
	defer service.mutex.Unlock()

	service.themes = result

	return nil
}

// loadModel loads a theme.json file from the filesystem and returns a Theme object
func (service *Theme) loadModel(themeID string, filesystem fs.FS) (model.Theme, error) {

	const location = "service.Theme.loadModel"
	result := model.NewTheme(themeID, service.funcMap)

	// Try to Oopen the theme.json file
	file, err := filesystem.Open("theme.json")

	if err != nil {
		return result, derp.Wrap(err, location, "Unable to open theme.json file", filesystem)
	}

	// Try to read the file into a buffer
	var buffer bytes.Buffer

	if _, err := io.Copy(&buffer, file); err != nil {
		return result, derp.Wrap(err, location, "Unable to read theme.json file", filesystem)
	}

	// Try to parse the JSON in the buffer into a Theme object
	if err := json.Unmarshal(buffer.Bytes(), &result); err != nil {
		return result, derp.Wrap(err, location, "Unable to parse theme.json file", filesystem)
	}

	// Success!
	return result, nil
}
