package service

import (
	"fmt"
	"html/template"
	"io/fs"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/slice"
)

// Layout service manages the global site layout that is stored in a particular path of the
// filesystem.
type Layout struct {
	filesystemService Filesystem
	funcMap           template.FuncMap
	locations         []config.Folder
	appearance        model.Layout
	connections       model.Layout
	domain            model.Layout
	global            model.Layout
	group             model.Layout
	navigation        model.Layout
	user              model.Layout

	changed chan bool
	closed  chan bool
}

// NewLayout returns a fully initialized Layout service.
func NewLayout(filesystemService Filesystem, funcMap template.FuncMap, locations []config.Folder) Layout {

	service := Layout{
		filesystemService: filesystemService,
		funcMap:           funcMap,
		changed:           make(chan bool),
		closed:            make(chan bool),
	}

	service.Refresh(locations)

	return service
}

/******************************************
 * Lifecycle Methods
 ******************************************/

func (service *Layout) Refresh(locations []config.Folder) {

	// If nothing has changed since the last time we refreshed, then we're done
	if slice.Equal(locations, service.locations) {
		return
	}

	fmt.Println("Refreshing layout")
	service.locations = locations

	// Try to load layouts from the filesystems
	if err := service.loadLayouts(); err != nil {
		derp.Report(err)
	}

	go service.Watch()
}

/******************************************
 * REAL TIME UPDATES
 ******************************************/

// watch must be run as a goroutine, and constantly monitors the
// "Updates" channel for news that a template has been updated.
func (service *Layout) Watch() {

	// Close the channel, which will stopp any existing watchers
	close(service.closed)

	// Create a new "closed" channel to close future watchers
	service.closed = make(chan bool)

	// Start new watchers.
	for _, folder := range service.locations {

		// for _, filename := range service.fileNames() {
		service.filesystemService.Watch(folder, service.changed, service.closed) // Fail silently because many locations may not define all layouts
		// }
	}

	// All Watchers Started.  Now Listen for Changes
	for {

		select {

		case <-service.changed:
			service.loadLayouts()

		case <-service.closed:
			return
		}
	}
}

// loadLayouts retrieves the template from the disk and parses it into
func (service *Layout) loadLayouts() error {

	// For each configured location...
	for _, location := range service.locations {

		// Get a valid filesystem adapter
		filesystem, err := service.filesystemService.GetFS(location)

		if err != nil {
			derp.Report(derp.Wrap(err, "service.layout.loadLayouts", "Unable to get filesystem adapter", location))
			continue // If there's an error, it means that this location just doesn't define this part of the layout.  It's OK
		}

		// Check each of the standard filenames
		for _, filename := range service.fileNames() {

			subFilesystem, err := fs.Sub(filesystem, filename)

			if err != nil {
				continue // If there's an error, it means that this location just doesn't define this part of the layout.  It's OK
			}

			fmt.Println("... layout: " + filename)

			layout := model.NewLayout(filename, service.funcMap)

			// System locations (except for "static" and "global") have a schema.json file
			if filename != "global" {
				if err := loadModelFromFilesystem(subFilesystem, &layout, filename); err != nil {
					return derp.Wrap(err, "service.layout.loadFromFilesystem", "Error loading Schema", location, filename)
				}
			}

			if err := loadHTMLTemplateFromFilesystem(subFilesystem, layout.HTMLTemplate, service.funcMap); err != nil {
				return derp.Wrap(err, "service.layout.loadFromFilesystem", "Error loading Template", location, filename)
			}

			if err := service.setLayout(filename, layout); err != nil {
				return derp.Wrap(err, "service.layout.loadFromFilesystem", "Error setting Layout", location, filename)
			}
		}
	}

	// Validate required fields.

	if service.appearance.HTMLTemplate == nil {
		return derp.NewInternalError("service.layout.loadFromFilesystem", "Appearance layout could not be loaded from any location", service.locations)
	}

	if service.connections.HTMLTemplate == nil {
		return derp.NewInternalError("service.layout.loadFromFilesystem", "Connections layout could not be loaded from any location", service.locations)
	}

	if service.domain.HTMLTemplate == nil {
		return derp.NewInternalError("service.layout.loadFromFilesystem", "Domain layout could not be loaded from any location", service.locations)
	}

	if service.global.HTMLTemplate == nil {
		return derp.NewInternalError("service.layout.loadFromFilesystem", "Global layout could not be loaded from any location", service.locations)
	}

	if service.group.HTMLTemplate == nil {
		return derp.NewInternalError("service.layout.loadFromFilesystem", "Group layout could not be loaded from any location", service.locations)
	}

	if service.navigation.HTMLTemplate == nil {
		return derp.NewInternalError("service.layout.loadFromFilesystem", "Top Level layout could not be loaded from any location", service.locations)
	}

	if service.user.HTMLTemplate == nil {
		return derp.NewInternalError("service.layout.loadFromFilesystem", "User layout could not be loaded from any location", service.locations)
	}

	return nil
}

/******************************************
 * HELPER METHODS
 ******************************************/

// loadFromFilesystem retrieves the template from the disk and parses it into
func (service *Layout) setLayout(name string, layout model.Layout) error {

	switch name {

	case "appearance":
		service.appearance = layout
		return nil

	case "connections":
		service.connections = layout
		return nil

	case "domain":
		service.domain = layout
		return nil

	case "global":
		service.global = layout
		return nil

	case "groups":
		service.group = layout
		return nil

	case "navigation":
		service.navigation = layout
		return nil

	case "users":
		service.user = layout
		return nil
	}

	return derp.NewInternalError("service.layout.setLayouts", "Invalid layout name", name)
}

// fileNames returns a list of directories that are owned by the Layout service.
func (service *Layout) fileNames() []string {
	return []string{"appearance", "connections", "domain", "global", "groups", "navigation", "users"}
}
