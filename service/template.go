package service

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"sort"
	"strings"
	"sync"

	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/list"
	"github.com/benpate/path"
	"github.com/benpate/schema"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/afero"
	"github.com/whisperverse/whisperverse/config"
	"github.com/whisperverse/whisperverse/model"
)

// Template service manages all of the templates in the system, and merges them with data to form fully populated HTML pages.
type Template struct {
	templates         map[string]model.Template // map of all templates available within this domain
	folder            config.Folder             // Configuration for template directory
	funcMap           template.FuncMap          // Map of functions to use in golang templates
	mutex             sync.RWMutex              // Mutext that locks access to the templates structure
	filesystem        afero.Fs                  // Filesystem where templates are stored.
	layoutService     *Layout                   // Pointer to the Layout service
	templateUpdateOut chan string               // Channel to notify other processes that a template has changed.
}

// NewTemplate returns a fully initialized Template service.
func NewTemplate(layoutService *Layout, funcMap template.FuncMap, folder config.Folder, templateUpdateChannel chan string) Template {

	return Template{
		templates:         make(map[string]model.Template),
		folder:            folder,
		funcMap:           funcMap,
		filesystem:        GetFS(folder),
		layoutService:     layoutService,
		templateUpdateOut: templateUpdateChannel,
	}
}

/*******************************************
 * COMMON DATA FUNCTIONS
 *******************************************/

// List returns all templates that match the provided criteria
func (service *Template) List(criteria exp.Expression) []model.Option {

	result := []model.Option{}

	for _, template := range service.templates {
		if path.Match(&template, criteria) {
			result = append(result, model.Option{
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

	service.mutex.Lock()
	defer service.mutex.Unlock()
	service.templates[template.TemplateID] = *template
	return nil
}

/*******************************************
 * CUSTOM QUERIES
 *******************************************/

// ListByContainer returns all model.Templates that match the provided "containedBy" value
func (service *Template) ListByContainer(containedBy string) []model.Option {
	return service.List(exp.Contains("containedBy", containedBy))
}

// ListByContainerLimited returns all model.Templates that match the provided "containedBy" value AND
// are present in the "limited" list.  If the "limited" list is empty, then all otherwise-valid templates
// are returned.
func (service *Template) ListByContainerLimited(containedBy string, limits []string) []model.Option {

	if len(limits) == 0 {
		return service.ListByContainer(containedBy)
	}

	return service.List(
		exp.And(
			exp.Contains("containedBy", containedBy),
			exp.ContainedBy("templateId", limits),
		),
	)
}

/*******************************************
 * OTHER DATA ACCESS FUNCTIONS
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

/*******************************************
 * REAL-TIME UPDATES
 *******************************************/

// watch must be run as a goroutine, and constantly monitors the
// "Updates" channel for news that a template has been updated.
func (service *Template) Watch() error {

	// Only synchronize on folders that are configured to do so.
	if !service.folder.Sync {
		return nil
	}

	// Only synchronize on FILESYSTEM folders (for now)
	if service.folder.Adapter != "FILE" {
		return nil
	}

	fmt.Println("Template service is loading templates:")

	// Load all templates from the filesystem
	fileList, err := ioutil.ReadDir(service.folder.Location)

	if err != nil {
		return derp.Wrap(err, "service.templateSource.File.List", "Unable to list files in filesystem", service.folder)
	}

	// Use a separate counter because not all files will be included in the result
	for _, fileInfo := range fileList {

		if fileInfo.IsDir() {
			templateID := list.Last(fileInfo.Name(), "/")

			fmt.Println(".. : " + templateID)
			// Add all other directories into the Template service as Templates
			template, err := service.loadFromFilesystem(templateID)

			if err != nil {
				derp.Report(derp.Wrap(err, "service.Template", "Error loading template from filesystem", templateID))
				continue
			}

			// Success!
			if err := service.Save(&template); err != nil {
				derp.Report(derp.Wrap(err, "service.Template", "Error saving Template to TemplateService", templateID))
				continue
			}
		}
	}

	go func() {

		fmt.Println("Watching for changes to " + service.folder.Location)

		// Create a new directory watcher
		watcher, err := fsnotify.NewWatcher()

		if err != nil {
			panic(err)
		}

		// List all the files in the directory
		files, err := ioutil.ReadDir(service.folder.Location)

		if err != nil {
			panic(err)
		}

		// Add watchers to each file that is a Directory.
		for _, file := range files {
			if file.IsDir() {
				if err := watcher.Add(service.folder.Location + "/" + file.Name()); err != nil {
					derp.Report(derp.Wrap(err, "service.Template.watch", "Error adding file watcher to file", file.Name()))
				}
			}
		}

		// Repeat indefinitely, listen and process file updates
		for {

			select {

			case event, ok := <-watcher.Events:

				if !ok {
					continue
				}

				// Only update when an HTML or JSON file has been changed
				if !(strings.HasSuffix(event.Name, ".html") || strings.HasSuffix(event.Name, ".json")) {
					continue
				}

				templateName := list.Last(list.RemoveLast(event.Name, "/"), "/")

				// Otherwise, add this folder to the Template service
				template, err := service.loadFromFilesystem(templateName)

				if err != nil {
					derp.Report(derp.Wrap(err, "service.Template.watch", "Error loading changes to template", event, templateName))
					continue
				}

				// Save the Template and notify Streams to update.
				if err := service.Save(&template); err != nil {
					derp.Report(derp.Wrap(err, "service.Template.watch", "Error saving changes to template", event, templateName))
					continue
				}

				// Temporarily removed when templates were moved into from "domain" scope into "server" scope
				// service.templateUpdateOut <- template.TemplateID

				fmt.Println("Updated template: " + template.Label)

			case err, ok := <-watcher.Errors:

				if ok {
					derp.Report(derp.Wrap(err, "service.Template.watch", "Error watching filesystem"))
				}
			}
		}
	}()

	return nil
}

/*******************************************
 * UTILITIES
 *******************************************/

// loadFromFilesystem locates and parses a Template sub-directory within the filesystem path
func (service *Template) loadFromFilesystem(templateID string) (model.Template, error) {

	filesystem := GetFS(service.folder, templateID)

	result := model.NewTemplate(templateID, service.funcMap)

	if err := loadModelFromFilesystem(filesystem, &result); err != nil {
		return result, derp.Wrap(err, "service.Template.loadFromFilesystem", "Error loading schema")
	}

	if err := loadHTMLTemplateFromFilesystem(filesystem, result.HTMLTemplate, service.funcMap); err != nil {
		return result, derp.Wrap(err, "service.Template.loadFromFilesystem", "Error loading template")
	}

	result.Validate()

	// Save the Template into the memory cache
	service.Save(&result)

	// Return to caller.
	return result, nil
}
