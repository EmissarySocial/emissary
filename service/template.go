package service

import (
	"html/template"
	"sort"
	"strings"
	"sync"

	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/path"
	"github.com/benpate/schema"

	"io/ioutil"

	"github.com/benpate/list"
	"github.com/fsnotify/fsnotify"
)

// Template service manages all of the templates in the system, and merges them with data to form fully populated HTML pages.
type Template struct {
	templates map[string]model.Template // map of all templates available within this domain
	funcMap   template.FuncMap          // Map of functions to use in golang templates
	mutex     sync.RWMutex              // Mutext that locks access to the templates structure
	path      string                    // Filesystem path to the template directory

	layoutService     *Layout
	layoutUpdates     chan bool
	templateUpdateOut chan string
}

// NewTemplate returns a fully initialized Template service.
func NewTemplate(domainService *Domain, layoutService *Layout, funcMap template.FuncMap, path string, layoutUpdates chan bool, templateUpdateChannel chan string) *Template {

	service := &Template{
		templates:         make(map[string]model.Template),
		funcMap:           funcMap,
		path:              path,
		layoutService:     layoutService,
		layoutUpdates:     layoutUpdates,
		templateUpdateOut: templateUpdateChannel,
	}

	// Load all templates from the filesystem
	list, err := ioutil.ReadDir(path)

	if err != nil {
		derp.Report(derp.Wrap(err, "ghost.service.templateSource.File.List", "Unable to list files in filesystem", path))
		return service
	}

	// Use a separate counter because not all files will be included in the result
	for _, fileInfo := range list {

		if fileInfo.IsDir() {
			name := fileInfo.Name()

			// System directories are skipped.
			if name[0] == '_' {
				continue
			}

			// Add all other directories into the Template service as Templates
			template := model.NewTemplate(name)
			if err := service.loadFromFilesystem(&template); err == nil {
				service.Save(&template)
			}
		}
	}

	go service.watch()

	return service
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
func (service *Template) Load(templateID string, template *model.Template) error {

	service.mutex.RLock()
	defer service.mutex.RUnlock()

	// Look in the local cache first
	if result, ok := service.templates[templateID]; ok {
		*template = result
		return nil
	}

	return derp.New(404, "ghost.sevice.Template.Load", "Template not found", templateID)
}

// Save adds/updates an Template in the memory cache
func (service *Template) Save(template *model.Template) error {

	service.mutex.Lock()
	defer service.mutex.Unlock()

	delete(service.templates, template.TemplateID)
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

	template := model.NewTemplate(templateID)

	// Try to find the Template used by this Stream
	if err := service.Load(templateID, &template); err != nil {
		return model.State{}, derp.Wrap(err, "ghost.service.Template.State", "Invalid Template", templateID)
	}

	// Try to find the state data for the state that the stream is in
	state, ok := template.State(stateID)

	if !ok {
		return state, derp.New(500, "ghost.service.Template.State", "Invalid state", templateID, stateID)
	}

	// Success!
	return state, nil
}

// Schema returns the Schema associated with this Stream
func (service *Template) Schema(templateID string) (schema.Schema, error) {

	template := model.NewTemplate(templateID)

	// Try to locate the Template used by this Stream
	if err := service.Load(templateID, &template); err != nil {
		return schema.Schema{}, derp.Wrap(err, "ghost.service.Template.Action", "Invalid Template", templateID)
	}

	// Return the Schema defined in this template.
	return template.Schema, nil
}

// ActionConfig returns the action definition that matches the stream and type provided
func (service *Template) Action(templateID string, actionID string) (model.Action, error) {

	template := model.NewTemplate(templateID)

	// Try to find the Template used by this Stream
	if err := service.Load(templateID, &template); err != nil {
		return model.Action{}, derp.Wrap(err, "ghost.service.Template.Action", "Invalid Template", templateID)
	}

	// Try to find the action in the Template
	if action, ok := template.Action(actionID); ok {
		return action, nil
	}

	// Not Found :(
	return model.Action{}, derp.New(derp.CodeBadRequestError, "ghost.service.Template.Action", "Unrecognized action", templateID, actionID)
}

/*******************************************
 * REAL-TIME UPDATES
 *******************************************/

// watch must be run as a goroutine, and constantly monitors the
// "Updates" channel for news that a template has been updated.
func (service *Template) watch() {

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
				derp.Report(derp.Wrap(err, "ghost.service.Template.watch", "Error adding file watcher to file", file.Name()))
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

			if event.Op != fsnotify.Write {
				continue
			}

			templateName := list.Last(list.RemoveLast(event.Name, "/"), "/")

			// System folders have a "_" prefix
			if strings.HasPrefix(templateName, "_") {

				// Static files are not processed.  Skip and continue
				if templateName == "_static" {
					continue
				}

				// Otherwise, add this folder to the Layout Service

				// Reload layout
				if err := service.layoutService.getTemplateFromFilesystem(templateName); err != nil {
					derp.Report(derp.Wrap(err, "ghost.service.Template", "Error reloading Layout template"))
				}

				// Update all Templates with new layout code
				for templateID := range service.templates {
					service.templateUpdateOut <- templateID
				}

				continue
			}

			// Otherwise, add this folder to the Template service
			template := new(model.Template)
			if err := service.loadFromFilesystem(template); err != nil {
				derp.Report(derp.Wrap(err, "ghost.service.Template.watch", "Error loading changes to template", event, templateName))
				continue
			}

			// Save the Template and notify Streams to update.
			service.Save(template)
			service.templateUpdateOut <- template.TemplateID

		case err, ok := <-watcher.Errors:

			if ok {
				derp.Report(derp.Wrap(err, "ghost.service.Template.watch", "Error watching filesystem"))
			}
		}
	}
}

/*******************************************
 * UTILITIES
 *******************************************/

// loadFromFilesystem locates and parses a Template sub-directory within the filesystem path
func (service *Template) loadFromFilesystem(t *model.Template) error {

	directory := service.path + "/" + t.TemplateID

	if err := loadTemplateFromFilesystem(directory, service.funcMap, t.HTMLTemplate); err != nil {
		return derp.Wrap(err, "ghost.service.Template.loadFromFilesystem", "Error loading template", directory)
	}

	if err := loadSchemaFromFilesystem(directory, &t.Schema); err != nil {
		return derp.Wrap(err, "ghost.service.Template.loadFromFilesystem", "Error loading schema", directory)
	}

	t.Validate()

	// Save the Template into the memory cache
	service.Save(t)

	// Return to caller.
	return nil
}
