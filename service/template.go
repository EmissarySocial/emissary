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
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
)

// Template service manages all of the templates in the system, and merges them with data to form fully populated HTML pages.
type Template struct {
	templates         set.Map[model.Template] // map of all templates available within this domain
	locations         []config.Folder         // Configuration for template directory
	filesystemService Filesystem              // Filesystem service
	funcMap           template.FuncMap        // Map of functions to use in golang templates
	mutex             sync.RWMutex            // Mutext that locks access to the templates structure
	changed           chan bool               // Channel that is used to signal that a template has changed
	closed            chan bool               // Channel to notify the watcher to close/reset
}

// NewTemplate returns a fully initialized Template service.
func NewTemplate(filesystemService Filesystem, funcMap template.FuncMap, locations []config.Folder) *Template {

	service := Template{
		templates:         make(set.Map[model.Template]),
		locations:         make([]config.Folder, 0),
		filesystemService: filesystemService,
		funcMap:           funcMap,
		changed:           make(chan bool),
		closed:            make(chan bool),
	}

	service.Refresh(locations)

	return &service
}

/******************************************
 * Lifecycle Methods
 ******************************************/

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

/******************************************
 * REAL-TIME UPDATES
 ******************************************/

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

	result := set.NewMap[model.Template]()

	// For each configured location...
	for _, location := range service.locations {

		// Get a valid filesystem adapter
		filesystem, err := service.filesystemService.GetFS(location)

		if err != nil {
			derp.Report(derp.Wrap(err, "service.Template.loadTemplates", "Error getting filesystem adapter", location))
			continue
		}

		directories, err := fs.ReadDir(filesystem, ".")

		if err != nil {
			derp.Report(derp.Wrap(err, "service.Template.loadTemplates", "Error reading directory", location))
			continue
		}

		for _, directory := range directories {

			if !directory.IsDir() {
				continue
			}

			subdirectory, err := fs.Sub(filesystem, directory.Name())

			if err != nil {
				derp.Report(derp.Wrap(err, "service.Template.loadTemplates", "Error getting filesystem adapter for sub-directory", location))
				continue
			}

			template := model.NewTemplate(directory.Name(), service.funcMap)

			// System locations (except for "static" and "global") have a schema.json file
			if err := loadModelFromFilesystem(subdirectory, &template, directory.Name()); err != nil {
				derp.Report(derp.Wrap(err, "service.template.loadFromFilesystem", "Error loading Schema", location, directory))
				continue
			}

			if err := loadHTMLTemplateFromFilesystem(subdirectory, template.HTMLTemplate, service.funcMap); err != nil {
				derp.Report(derp.Wrap(err, "service.template.loadFromFilesystem", "Error loading Template", location, directory))
				continue
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

/******************************************
 * Common Data Methods
 ******************************************/

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
	return nil, derp.NewNotFoundError("sevice.Template.Load", "Template not found", templateID, keys)
}

/******************************************
 * Custom Queries
 ******************************************/

// ListFeatures returns all templates that are used as "feature" templates
func (service *Template) ListFeatures() []form.LookupCode {

	filter := func(template *model.Template) bool {
		return template.IsFeature()
	}

	return service.List(filter)
}

// ListByContainer returns all model.Templates that match the provided "containedByRole" value
func (service *Template) ListByContainer(containedByRole string) []form.LookupCode {

	filter := func(t *model.Template) bool {
		return t.ContainedBy.Contains(containedByRole)
	}

	return service.List(filter)
}

// ListByContainerLimited returns all model.Templates that match the provided "containedByRole" value AND
// are present in the "limited" list.  If the "limited" list is empty, then all otherwise-valid templates
// are returned.
func (service *Template) ListByContainerLimited(containedByRole string, limits sliceof.String) []form.LookupCode {

	if limits.IsEmpty() {
		return service.ListByContainer(containedByRole)
	}

	filter := func(t *model.Template) bool {
		return t.ContainedBy.Contains(containedByRole) && limits.Contains(t.TemplateID)
	}

	return service.List(filter)
}

/******************************************
 * Admin Templates
 ******************************************/

func (service *Template) LoadAdmin(templateID string) (*model.Template, error) {

	templateID = "admin-" + templateID

	// Try to load the template
	template, err := service.Load(templateID)

	if err != nil {
		return nil, derp.Wrap(err, "service.Template.LoadAdmin", "Unable to load admin template", templateID)
	}

	// RULE: Validate Template ContainedBy
	if template.Role != "admin" {
		return nil, derp.NewInternalError("service.Template.LoadAdmin", "Template must have 'admin' role.", template.TemplateID, template.Role)
	}

	if !template.ContainedBy.Equal([]string{"admin"}) {
		return nil, derp.NewInternalError("service.Template.LoadAdmin", "Template must be contained by 'admin'", template.TemplateID, template.ContainedBy)
	}

	// Success!
	return template, nil
}

/******************************************
 * Other Data Access Methods
 ******************************************/

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
		return state, derp.NewInternalError("service.Template.State", "Invalid state", templateID, stateID)
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
