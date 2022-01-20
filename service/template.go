package service

import (
	"html/template"
	"sort"
	"sync"

	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/list"
	"github.com/benpate/path"
	"github.com/benpate/schema"
	"github.com/whisperverse/whisperverse/model"

	"io/ioutil"

	"github.com/fsnotify/fsnotify"
)

// Template service manages all of the templates in the system, and merges them with data to form fully populated HTML pages.
type Template struct {
	templates map[string]model.Template // map of all templates available within this domain
	funcMap   template.FuncMap          // Map of functions to use in golang templates
	mutex     sync.RWMutex              // Mutext that locks access to the templates structure
	adapter   string                    // kind of filesystem adapter to use
	path      string                    // Filesystem path to the template directory

	layoutService     *Layout
	templateUpdateOut chan string
}

// NewTemplate returns a fully initialized Template service.
func NewTemplate(layoutService *Layout, funcMap template.FuncMap, adapter string, path string, templateUpdateChannel chan string) Template {

	return Template{
		templates:         make(map[string]model.Template),
		funcMap:           funcMap,
		adapter:           adapter,
		path:              path,
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

	return nil, derp.New(404, "whisper.sevice.Template.Load", "Template not found", templateID, service.templates)
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
		return model.State{}, derp.Wrap(err, "whisper.service.Template.State", "Invalid Template", templateID)
	}

	// Try to find the state data for the state that the stream is in
	state, ok := template.State(stateID)

	if !ok {
		return state, derp.New(500, "whisper.service.Template.State", "Invalid state", templateID, stateID)
	}

	// Success!
	return state, nil
}

// Schema returns the Schema associated with this Stream
func (service *Template) Schema(templateID string) (schema.Schema, error) {

	// Try to locate the Template used by this Stream
	template, err := service.Load(templateID)

	if err != nil {
		return schema.Schema{}, derp.Wrap(err, "whisper.service.Template.Action", "Invalid Template", templateID)
	}

	// Return the Schema defined in this template.
	return template.Schema, nil
}

// ActionConfig returns the action definition that matches the stream and type provided
func (service *Template) Action(templateID string, actionID string) (*model.Action, error) {

	// Try to find the Template used by this Stream
	template, err := service.Load(templateID)

	if err != nil {
		return nil, derp.Wrap(err, "whisper.service.Template.Action", "Invalid Template", templateID)
	}

	// Try to find the action in the Template
	action := template.Action(actionID)

	if action == nil {
		return action, derp.NewNotFoundError("whisper.service.Template.Action", "Unrecognized action", templateID, actionID)
	}

	return action, nil
}

/*******************************************
 * REAL-TIME UPDATES
 *******************************************/

// watch must be run as a goroutine, and constantly monitors the
// "Updates" channel for news that a template has been updated.
func (service *Template) Watch() error {

	// Load all templates from the filesystem
	fileList, err := ioutil.ReadDir(service.path)

	if err != nil {
		return derp.Wrap(err, "whisper.service.templateSource.File.List", "Unable to list files in filesystem", service.path)
	}

	go func() {
		// Use a separate counter because not all files will be included in the result
		for _, fileInfo := range fileList {

			if fileInfo.IsDir() {
				templateID := list.Last(fileInfo.Name(), "/")

				// System directories are skipped.
				if templateID == "system" {
					continue
				}

				// Add all other directories into the Template service as Templates
				template, err := service.loadFromFilesystem(templateID)

				if err != nil {
					derp.Report(derp.Wrap(err, "whisper.service.Template", "Error loading template from filesystem", templateID))
					continue
				}

				// Success!
				if err := service.Save(&template); err != nil {
					derp.Report(derp.Wrap(err, "whisper.service.Template", "Error saving Template to TemplateService", templateID))
					continue
				}
			}
		}

		// Create a new directory watcher
		watcher, err := fsnotify.NewWatcher()

		if err != nil {
			panic(err)
		}

		// List all the files in the directory
		files, err := ioutil.ReadDir(service.path)

		if err != nil {
			panic(err)
		}

		// Add watchers to each file that is a Directory.
		for _, file := range files {
			if file.IsDir() {
				if err := watcher.Add(service.path + "/" + file.Name()); err != nil {
					derp.Report(derp.Wrap(err, "whisper.service.Template.watch", "Error adding file watcher to file", file.Name()))
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

				templateName := list.Last(list.RemoveLast(event.Name, "/"), "/")

				// Static files are not processed.  Skip and continue
				if templateName == "system" {
					continue
				}

				// Otherwise, add this folder to the Template service
				template, err := service.loadFromFilesystem(templateName)

				if err != nil {
					derp.Report(derp.Wrap(err, "whisper.service.Template.watch", "Error loading changes to template", event, templateName))
					continue
				}

				// Save the Template and notify Streams to update.
				service.Save(&template)
				service.templateUpdateOut <- template.TemplateID

			case err, ok := <-watcher.Errors:

				if ok {
					derp.Report(derp.Wrap(err, "whisper.service.Template.watch", "Error watching filesystem"))
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

	directory := service.path + "/" + templateID
	result := model.NewTemplate(templateID, service.funcMap)

	if err := loadModelFromFilesystem(directory, &result); err != nil {
		return result, derp.Wrap(err, "whisper.service.Template.loadFromFilesystem", "Error loading schema", directory)
	}

	if err := loadHTMLTemplateFromFilesystem(directory, result.HTMLTemplate, service.funcMap); err != nil {
		return result, derp.Wrap(err, "whisper.service.Template.loadFromFilesystem", "Error loading template", directory)
	}

	result.Validate()

	// Save the Template into the memory cache
	service.Save(&result)

	// Return to caller.
	return result, nil
}
