package service

import (
	"fmt"
	"html/template"
	"io/fs"

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

	changed chan bool
	closed  chan bool
}

// NewTheme returns a fully initialized Theme service.
func NewTheme(filesystemService Filesystem, funcMap template.FuncMap, locations []config.Folder) Theme {

	service := Theme{
		filesystemService: filesystemService,
		funcMap:           funcMap,
		locations:         make([]config.Folder, 0),
		themes:            mapof.NewObject[model.Theme](),
		changed:           make(chan bool),
		closed:            make(chan bool),
	}

	service.Refresh(locations)

	return service
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

func (service *Theme) GetTheme(themeID string) model.Theme {

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
	// This should never happen
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

			themeName := themeDirectory.Name()
			fmt.Println("... theme: " + themeName)

			subFilesystem, err := fs.Sub(filesystem, themeName)

			if err != nil {
				fmt.Println("... error loading subdirectory.")
				continue // If there's an error, it means that this location just doesn't define this part of the theme.  It's OK
			}

			theme := model.NewTheme(themeName, service.funcMap)

			if err := loadHTMLTemplateFromFilesystem(subFilesystem, theme.HTMLTemplate, service.funcMap); err != nil {
				fmt.Println("... error reading files.")
				derp.Report(derp.Wrap(err, "service.theme.loadFromFilesystem", "Error loading Template", location, themeName))
				continue
			}

			fmt.Println(". Success.")

			service.themes[themeName] = theme
		}
	}

	return nil
}
