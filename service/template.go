package service

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"sort"
	"sync"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/compare"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/schema"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/afero"
)

// Template service manages all of the templates in the system, and merges them with data to form fully populated HTML pages.
type Template struct {
	templates     map[string]model.Template // map of all templates available within this domain
	folder        config.Folder             // Configuration for template directory
	funcMap       template.FuncMap          // Map of functions to use in golang templates
	mutex         sync.RWMutex              // Mutext that locks access to the templates structure
	filesystem    afero.Fs                  // Filesystem where templates are stored.
	layoutService *Layout                   // Pointer to the Layout service

	closeWatcher chan bool // Channel to notify the watcher to close/reset
}

// NewTemplate returns a fully initialized Template service.
func NewTemplate(layoutService *Layout, funcMap template.FuncMap, folder config.Folder) *Template {

	service := Template{
		templates:     make(map[string]model.Template),
		funcMap:       funcMap,
		filesystem:    afero.NewMemMapFs(),
		layoutService: layoutService,
	}

	service.Refresh(folder)

	return &service
}

/*******************************************
 * LIFECYCLE METHODS
 *******************************************/

func (service *Template) Refresh(folder config.Folder) {

	// RULE: If the Filesystem is empty, then don't try to load
	if service.folder.IsEmpty() {
		return
	}

	// RULE: If nothing has changed since the last time we refreshed, then we're done.
	if folder == service.folder {
		return
	}

	// Try to get a filesystem that matches this folder
	filesystem, err := GetFS(folder)

	if err != nil {
		derp.Report(derp.Wrap(err, "service.Template", "Invalid filesystem", folder))
		return
	}

	// Add configuration to the service
	service.folder = folder
	service.filesystem = filesystem

	// Load all templates from the filesystem
	if err := service.loadTemplates(); err != nil {
		derp.Report(derp.Wrap(err, "service.Template.Refresh", "Error loading templates from filesystem"))
		return
	}

	// Try to watch the template directory for changes
	go service.watch()
}

/*******************************************
 * STARTUP METHODS
 *******************************************/

func (service *Template) loadTemplates() error {

	fmt.Println("Loading Templates from: " + service.folder.Location)

	// Load all templates from the filesystem
	fileList, err := ioutil.ReadDir(service.folder.Location)

	if err != nil {
		return derp.Wrap(err, "service.templateSource.File.List", "Unable to list files in filesystem", service.folder)
	}

	result := make(map[string]model.Template)

	// Use a separate counter because not all files will be included in the result
	for _, fileInfo := range fileList {

		if fileInfo.IsDir() {
			templateID := list.Slash(fileInfo.Name()).Last()

			fmt.Println(".. : " + templateID)

			// Add all other directories into the Template service as Templates
			template, err := service.loadFromFilesystem(templateID)

			if err != nil {
				derp.Report(derp.Wrap(err, "service.Template", "Error loading template from filesystem", templateID))
				continue
			}

			// Put the template into the temporary map.  This will be switched into the templates map once all templates are loaded.
			result[templateID] = template
		}
	}

	// Lock service and update templates
	service.mutex.Lock()
	defer service.mutex.Unlock()

	service.templates = result

	// We made it!  Success :)
	return nil
}

// loadFromFilesystem locates and parses a Template sub-directory within the filesystem path
func (service *Template) loadFromFilesystem(templateID string) (model.Template, error) {

	const location = "service.Template.loadFromFilesystem"

	result := model.NewTemplate(templateID, service.funcMap)

	filesystem, err := GetFS(service.folder, templateID)

	if err != nil {
		return result, derp.Wrap(err, location, "Unable to get filesystem for template", service.folder, templateID)
	}

	if err := loadModelFromFilesystem(filesystem, &result); err != nil {
		return result, derp.Wrap(err, location, "Error loading schema")
	}

	if err := loadHTMLTemplateFromFilesystem(filesystem, result.HTMLTemplate, service.funcMap); err != nil {
		return result, derp.Wrap(err, location, "Error loading template")
	}

	// RULE: If this is a "feature" template, then the default action is always "feature"
	if result.AsFeature {
		result.DefaultAction = "feature"
	}

	// RULE: Validate that the default action is not nil
	if result.Default() == nil {
		return result, derp.NewInternalError(location, "Invalid Template: Missing 'default' method", templateID)
	}

	// Return to caller.
	return result, nil
}

/*******************************************
 * REAL-TIME UPDATES
 *******************************************/

// watch must be run as a goroutine, and constantly monitors the
// "Updates" channel for news that a template has been updated.
func (service *Template) watch() {

	// abort the existing watcher
	close(service.closeWatcher)

	// RULE: Only synchronize on folders that are configured to do so.
	if !service.folder.Sync {
		return
	}

	// RULE: Only synchronize on FILESYSTEM folders (for now)
	if service.folder.Adapter != "FILE" {
		return
	}

	// OK, let's do this.
	fmt.Println("Watching for changes to " + service.folder.Location)

	// Initialize a new channel to close this watcher when we need to.
	service.closeWatcher = make(chan bool)

	// List all the files in the directory
	files, err := ioutil.ReadDir(service.folder.Location)

	if err != nil {
		derp.Report(derp.Wrap(err, "service.Template", "Error listing files in filesystem", service.folder.Location))
		return
	}

	// Create a new directory watcher
	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		derp.Report(derp.Wrap(err, "service.Template", "Error creating filesystem watcher", service.folder.Location))
		return
	}

	defer watcher.Close()

	// Add watchers to each file that is a Directory.
	for _, file := range files {
		if file.IsDir() {
			if err := watcher.Add(service.folder.Location + "/" + file.Name()); err != nil {
				derp.Report(derp.Wrap(err, "service.Template.watch", "Error adding file watcher to file", file.Name()))
				// Do not return if there is an error here.  We want to keep watching for other changes.
			}
		}
	}

	// Repeat indefinitely, listen and process file updates
	for {

		select {

		case <-watcher.Events:
			if err := service.loadTemplates(); err != nil {
				derp.Report(derp.Wrap(err, "service.Template.watch", "Error loading templates from filesystem"))
			}

		case err := <-watcher.Errors:
			derp.Report(derp.Wrap(err, "service.Template.watch", "Error watching filesystem"))

		case <-service.closeWatcher:
			return
		}
	}
}

/*******************************************
 * COMMON DATA METHODS
 *******************************************/

// List returns all templates that match the provided criteria
func (service *Template) List(filter func(*model.Template) bool) []form.OptionCode {

	result := []form.OptionCode{}

	for _, template := range service.templates {
		if filter(&template) {
			result = append(result, form.OptionCode{
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

// Save adds/updates an Template in the memory cache
func (service *Template) Save(template *model.Template) error {

	// WRITE Mutex to make multi-threaded access safe.
	service.mutex.Lock()
	defer service.mutex.Unlock()

	// Do the thing.
	service.templates[template.TemplateID] = *template
	return nil
}

/*******************************************
 * CUSTOM QUERIES
 *******************************************/

// ListFeatures returns all templates that are used as "feature" templates
func (service *Template) ListFeatures() []form.OptionCode {

	filter := func(t *model.Template) bool {
		return t.AsFeature
	}

	return service.List(filter)
}

// ListByContainer returns all model.Templates that match the provided "containedBy" value
func (service *Template) ListByContainer(containedBy string) []form.OptionCode {

	filter := func(t *model.Template) bool {
		return compare.Contains(t.ContainedBy, containedBy)
	}

	return service.List(filter)
}

// ListByContainerLimited returns all model.Templates that match the provided "containedBy" value AND
// are present in the "limited" list.  If the "limited" list is empty, then all otherwise-valid templates
// are returned.
func (service *Template) ListByContainerLimited(containedBy string, limits []string) []form.OptionCode {

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
