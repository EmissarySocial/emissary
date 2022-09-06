package service

import (
	"fmt"
	"html/template"
	"io/fs"
	"sort"
	"sync"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/compare"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
)

// Template service manages all of the templates in the system, and merges them with data to form fully populated HTML pages.
type Template struct {
	templates         set.Map[string, model.Template] // map of all templates available within this domain
	locations         []config.Folder                 // Configuration for template directory
	funcMap           template.FuncMap                // Map of functions to use in golang templates
	mutex             sync.RWMutex                    // Mutext that locks access to the templates structure
	templateService   *Layout                         // Pointer to the Layout service
	filesystemService Filesystem                      // Filesystem service

	changed chan bool // Channel that is used to signal that a template has changed
	closed  chan bool // Channel to notify the watcher to close/reset
}

// NewTemplate returns a fully initialized Template service.
func NewTemplate(templateService *Layout, filesystemService Filesystem, funcMap template.FuncMap, locations []config.Folder) *Template {

	service := Template{
		templates:         make(set.Map[string, model.Template]),
		funcMap:           funcMap,
		locations:         make([]config.Folder, 0),
		templateService:   templateService,
		filesystemService: filesystemService,
		changed:           make(chan bool),
		closed:            make(chan bool),
	}

	service.Refresh(locations)

	return &service
}

/*******************************************
 * LIFECYCLE METHODS
 *******************************************/

func (service *Template) Refresh(locations []config.Folder) {

	// RULE: If the Filesystem is empty, then don't try to load
	if len(locations) == 0 {
		return
	}

	// RULE: If nothing has changed since the last time we refreshed, then we're done.
	if slice.Equal(locations, service.locations) {
		return
	}

	// Add configuration to the service
	service.locations = locations

	// Load all templates from the filesystem
	if err := service.loadTemplates(); err != nil {
		derp.Report(derp.Wrap(err, "service.Template.Refresh", "Error loading templates from filesystem"))
		return
	}

	// Try to watch the template directory for changes
	go service.watch()
}

/*******************************************
 * REAL-TIME UPDATES
 *******************************************/

// watch must be run as a goroutine, and constantly monitors the
// "Updates" channel for news that a template has been updated.
func (service *Template) watch() {

	// abort the existing watcher
	close(service.closed)

	// open a new channel for the next watcher
	service.closed = make(chan bool)

	// Start new watchers.
	for _, folder := range service.locations {

		if err := service.filesystemService.Watch(folder, service.changed, service.closed); err != nil {
			derp.Report(derp.Wrap(err, "service.Layout.Watch", "Error watching filesystem", folder))
		}
	}

	// All Watchers Started.  Now Listen for Changes
	for {

		select {

		case <-service.changed:
			service.loadTemplates()

		case <-service.closed:
			return
		}
	}
}

// loadTemplates retrieves the template from the filesystem and parses it into
func (service *Template) loadTemplates() error {

	result := make(set.Map[string, model.Template])

	// For each configured location...
	for _, location := range service.locations {

		// Get a valid filesystem adapter
		filesystem, err := service.filesystemService.GetFS(location)

		if err != nil {
			return derp.Wrap(err, "service.Template.loadTemplates", "Error getting filesystem adapter", location)
		}

		directories, err := fs.ReadDir(filesystem, ".")

		if err != nil {
			return derp.Wrap(err, "service.Template.loadTemplates", "Error reading directory", location)
		}

		for _, directory := range directories {

			if !directory.IsDir() {
				continue
			}

			subdirectory, err := fs.Sub(filesystem, directory.Name())

			if err != nil {
				return derp.Wrap(err, "service.Template.loadTemplates", "Error getting filesystem adapter for sub-directory", location)
			}

			template := model.NewTemplate(directory.Name(), service.funcMap)

			// System locations (except for "static" and "global") have a schema.json file
			if err := loadModelFromFilesystem(subdirectory, &template); err != nil {
				return derp.Wrap(err, "service.template.loadFromFilesystem", "Error loading Schema", location, directory)
			}

			if err := loadHTMLTemplateFromFilesystem(subdirectory, template.HTMLTemplate, service.funcMap); err != nil {
				return derp.Wrap(err, "service.template.loadFromFilesystem", "Error loading Template", location, directory)
			}

			fmt.Println("... template: " + template.TemplateID)

			result[template.TemplateID] = template
		}
	}

	// Lock service and update templates
	service.mutex.Lock()
	defer service.mutex.Unlock()

	service.templates = result

	return nil
}

/*******************************************
 * COMMON DATA METHODS
 *******************************************/

// List returns all templates that match the provided criteria
func (service *Template) List(filter func(*model.Template) bool) []form.LookupCode {

	result := []form.LookupCode{}

	for _, template := range service.templates {
		if filter(&template) {
			result = append(result, form.LookupCode{
				Value:       template.TemplateID,
				Label:       template.Label,
				Description: template.Description,
				Icon:        template.Icon,
				Group:       template.Category,
			})
		}
	}

	// Sort templates by Group, then Label
	sort.Slice(result, func(a int, b int) bool {
		if result[a].Group == result[b].Group {
			return result[a].Label < result[b].Label
		}
		return result[a].Group < result[b].Group
	})

	return result
}

// Load retrieves an Template from the database
func (service *Template) Load(templateID string) (*model.Template, error) {

	// READ Mutex to make multi-threaded access safe.
	service.mutex.RLock()
	defer service.mutex.RUnlock()

	// Look in the local cache first
	if template, ok := service.templates[templateID]; ok {
		return &template, nil
	}

	// Collect keys for error report:
	keys := make([]string, 0)
	for key := range service.templates {
		keys = append(keys, key)
	}
	return nil, derp.New(404, "sevice.Template.Load", "Template not found", templateID, keys)
}

/*******************************************
 * CUSTOM QUERIES
 *******************************************/

// ListFeatures returns all templates that are used as "feature" templates
func (service *Template) ListFeatures() []form.LookupCode {

	filter := func(t *model.Template) bool {
		return t.AsFeature
	}

	return service.List(filter)
}

// ListByContainer returns all model.Templates that match the provided "containedBy" value
func (service *Template) ListByContainer(containedBy string) []form.LookupCode {

	filter := func(t *model.Template) bool {
		return compare.Contains(t.ContainedBy, containedBy)
	}

	return service.List(filter)
}

// ListByContainerLimited returns all model.Templates that match the provided "containedBy" value AND
// are present in the "limited" list.  If the "limited" list is empty, then all otherwise-valid templates
// are returned.
func (service *Template) ListByContainerLimited(containedBy string, limits []string) []form.LookupCode {

	if len(limits) == 0 {
		return service.ListByContainer(containedBy)
	}

	filter := func(t *model.Template) bool {
		return compare.Contains(t.ContainedBy, containedBy) && compare.Contains(limits, t.TemplateID)
	}

	return service.List(filter)
}

/*******************************************
 * OTHER DATA ACCESS METHODS
 *******************************************/

// State returns the detailed State information associated with this Stream
func (service *Template) State(templateID string, stateID string) (model.State, error) {

	// Try to find the Template used by this Stream
	template, err := service.Load(templateID)

	if err != nil {
		return model.State{}, derp.Wrap(err, "service.Template.State", "Invalid Template", templateID)
	}

	// Try to find the state data for the state that the stream is in
	state, ok := template.State(stateID)

	if !ok {
		return state, derp.New(500, "service.Template.State", "Invalid state", templateID, stateID)
	}

	// Success!
	return state, nil
}

// Schema returns the Schema associated with this Stream
func (service *Template) Schema(templateID string) (schema.Schema, error) {

	// Try to locate the Template used by this Stream
	template, err := service.Load(templateID)

	if err != nil {
		return schema.Schema{}, derp.Wrap(err, "service.Template.Action", "Invalid Template", templateID)
	}

	// Return the Schema defined in this template.
	return template.Schema, nil
}

// ActionConfig returns the action definition that matches the stream and type provided
func (service *Template) Action(templateID string, actionID string) (*model.Action, error) {

	// Try to find the Template used by this Stream
	template, err := service.Load(templateID)

	if err != nil {
		return nil, derp.Wrap(err, "service.Template.Action", "Invalid Template", templateID)
	}

	// Try to find the action in the Template
	action := template.Action(actionID)

	if action == nil {
		return action, derp.NewNotFoundError("service.Template.Action", "Unrecognized action", templateID, actionID)
	}

	return action, nil
}
