package service

import (
	"fmt"
	"html/template"
	"io/ioutil"

	"github.com/benpate/derp"
	"github.com/benpate/list"
	"github.com/fsnotify/fsnotify"
)

// Layout service manages the global site layout that is stored in a particular path of the
// filesystem.
type Layout struct {
	path     string
	template *template.Template
}

// NewLayout returns a fully initialized Layout service.
func NewLayout(path string, updates chan *template.Template) (*Layout, error) {

	layout := &Layout{
		path: path,
	}

	if err := layout.Load(); err != nil {
		return layout, derp.Wrap(err, "ghost.service.NewLayout", "Error creating new Layout service", path)
	}

	go layout.start(updates)

	return layout, nil
}

// Layout returns a clone of the current layout template.
func (service *Layout) Layout() *template.Template {
	result, err := service.template.Clone()
	derp.Report(err)
	return result
}

// Load retrieves the template from the disk and parses it into
func (service *Layout) Load() error {

	// Create a new template.Template to return on success
	layout := template.New("")

	// Try to list all files in the configured path
	files, err := ioutil.ReadDir(service.path)

	if err != nil {
		return derp.Wrap(err, "ghost.service.NewLayout", "Unable to list files in filesystem")
	}

	for _, file := range files {

		// Do not parse directory trees
		if file.IsDir() {
			continue
		}

		// Read template file contents into memory
		contents, err := ioutil.ReadFile(service.path + "/" + file.Name())

		if err != nil {
			return derp.Wrap(err, "ghost.service.NewLayout", "Error reading file from filesystem", file.Name())
		}

		// Parse the current file into an HTML template
		t, err := template.New("").Parse(string(contents))

		if err != nil {
			return derp.Wrap(err, "ghost.service.NewLayout", "Error parsing template file", file.Name(), string(contents))
		}

		// Try to append this layout to the ParseTree
		templateName := list.Head(file.Name(), ".")
		if _, err := layout.AddParseTree(templateName, t.Tree); err != nil {
			return derp.Wrap(err, "ghost.service.NewLayout", "Error adding parseTree")
		}
	}

	// Success!!
	service.template = layout
	fmt.Println("updated layout service with new layout: ")

	return nil
}

// watch is called by the "NewLayout" initializer.  This method creates its own watcher
// on the path that contains the layout.  Any updates to the layout files will reload the
// layout Template, and push it into the Updates channel, so that any waiting pages can be
// refreshed dynamically.
func (service *Layout) start(updates chan *template.Template) error {

	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		return derp.Wrap(err, "ghost.service.Layout.Watch", "Could not watch filesystem")
	}

	if err := watcher.Add(service.path); err != nil {
		return derp.Wrap(err, "ghost.service.Layout.Watch", "Error adding watcher on path", service.path)
	}

	for {
		select {
		case event, ok := <-watcher.Events:

			if ok {
				if event.Op != fsnotify.Write {
					continue
				}

				if err := service.Load(); err != nil {
					derp.Report(derp.Wrap(err, "ghost.service.Layout.Watch", "Error re-loading Layout"))
					continue
				}

				updates <- service.template
			}

		case err, ok := <-watcher.Errors:

			if ok {
				derp.Report(derp.Wrap(err, "ghost.service.Layout.Watch", "Error watching filesystem"))
			}
		}
	}
}
