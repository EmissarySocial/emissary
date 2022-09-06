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
	analytics         model.Layout
	appearance        model.Layout
	connections       model.Layout
	domain            model.Layout
	global            model.Layout
	group             model.Layout
	profile           model.Layout
	topLevel          model.Layout
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

/*******************************************
 * LIFECYCLE METHODS
 *******************************************/

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

/*******************************************
 * REAL TIME UPDATES
 *******************************************/

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
				if err := loadModelFromFilesystem(subFilesystem, &layout); err != nil {
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
	if service.analytics.HTMLTemplate == nil {
		return derp.NewInternalError("service.layout.loadFromFilesystem", "Error Analytics Template could not be loaded from any location", service.locations)
	}

	if service.appearance.HTMLTemplate == nil {
		return derp.NewInternalError("service.layout.loadFromFilesystem", "Error Appearance Template could not be loaded from any location", service.locations)
	}

	if service.connections.HTMLTemplate == nil {
		return derp.NewInternalError("service.layout.loadFromFilesystem", "Error Connections Template could not be loaded from any location", service.locations)
	}

	if service.domain.HTMLTemplate == nil {
		return derp.NewInternalError("service.layout.loadFromFilesystem", "Error Domain Template could not be loaded from any location", service.locations)
	}

	if service.global.HTMLTemplate == nil {
		return derp.NewInternalError("service.layout.loadFromFilesystem", "Error Global Template could not be loaded from any location", service.locations)
	}

	if service.group.HTMLTemplate == nil {
		return derp.NewInternalError("service.layout.loadFromFilesystem", "Error Group Template could not be loaded from any location", service.locations)
	}

	if service.profile.HTMLTemplate == nil {
		return derp.NewInternalError("service.layout.loadFromFilesystem", "Error Profile Template could not be loaded from any location", service.locations)
	}

	if service.topLevel.HTMLTemplate == nil {
		return derp.NewInternalError("service.layout.loadFromFilesystem", "Error Top Level Template could not be loaded from any location", service.locations)
	}

	if service.user.HTMLTemplate == nil {
		return derp.NewInternalError("service.layout.loadFromFilesystem", "Error User Template could not be loaded from any location", service.locations)
	}

	return nil
}

/*******************************************
 * LAYOUT ACCESSORS
 *******************************************/

func (service *Layout) Analytics() *model.Layout {
	return &service.analytics
}

func (service *Layout) Appearance() *model.Layout {
	return &service.appearance
}

func (service *Layout) Connections() *model.Layout {
	return &service.connections
}

func (service *Layout) Domain() *model.Layout {
	return &service.domain
}

func (service *Layout) Global() *model.Layout {
	return &service.global
}

func (service *Layout) Group() *model.Layout {
	return &service.group
}

func (service *Layout) Profile() *model.Layout {
	return &service.profile
}

func (service *Layout) TopLevel() *model.Layout {
	return &service.topLevel
}

func (service *Layout) User() *model.Layout {
	return &service.user
}

/*******************************************
 * HELPER METHODS
 *******************************************/

// loadFromFilesystem retrieves the template from the disk and parses it into
func (service *Layout) setLayout(name string, layout model.Layout) error {

	switch name {

	case "analytics":
		service.analytics = layout
		return nil

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

	case "profiles":
		service.profile = layout
		return nil

	case "toplevel":
		service.topLevel = layout
		return nil

	case "users":
		service.user = layout
		return nil
	}

	return derp.NewInternalError("service.layout.setLayouts", "Invalid layout name", name)
}

// fileNames returns a list of directories that are owned by the Layout service.
func (service *Layout) fileNames() []string {
	return []string{"analytics", "appearance", "connections", "domain", "global", "groups", "profiles", "toplevel", "users"}
}
