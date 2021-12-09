package service

import (
	"html/template"
	"sync"

	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/schema"

	"encoding/json"
	"io/ioutil"

	"github.com/benpate/list"
	"github.com/fsnotify/fsnotify"

	minify "github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
)

// Template service manages all of the templates in the system, and merges them with data to form fully populated HTML pages.
type Template struct {
	templates map[string]model.Template // map of all templates available within this domain
	funcMap   template.FuncMap          // Map of functions to use in golang templates
	mutex     sync.RWMutex              // Mutext that locks access to the templates structure
	path      string                    // Filesystem path to the template directory

	layoutService     *Layout
	layoutUpdates     chan bool
	templateUpdateOut chan model.Template
}

// NewTemplate returns a fully initialized Template service.
func NewTemplate(path string, layoutService *Layout, funcMap template.FuncMap, layoutUpdates chan bool, templateUpdateChannel chan model.Template) *Template {

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
			switch name {
			case "layout": // Skip "layout" folder because it's handled by the LayoutService
			case "static": // Skip "static" folder because it's served directly to the client

			default:
				// Add all other directories into the Template service as Templates
				template := model.NewTemplate(name)
				if err := service.loadFromFilesystem(&template); err == nil {
					service.Save(&template)
				}
			}
		}
	}

	go service.watch()

	return service
}

// loadFromFilesystem locates and parses a Template sub-directory within the filesystem path
func (service *Template) loadFromFilesystem(t *model.Template) error {

	directory := service.path + "/" + t.TemplateID
	files, err := ioutil.ReadDir(directory)

	if err != nil {
		return derp.Wrap(err, "ghost.service.Template.loadFromFilesystem", "Unable to list directory", directory)
	}

	// Create the minifier
	m := minify.New()
	m.AddFunc("text/html", html.Minify)

	// Load the schema.json file
	{
		content, err := ioutil.ReadFile(directory + "/schema.json")

		if err != nil {
			return derp.Wrap(err, "ghost.service.Template.loadFromFilesystem", "Cannot read file: schema.json", t.TemplateID)
		}

		if err := json.Unmarshal(content, t); err != nil {
			return derp.Wrap(err, "ghost.service.Template.loadFromFilesystem", "Invalid JSON configuration file: schema.json", t.TemplateID)
		}
	}

	// Views are processed FIRST so we can generate a list of objects to enter into the different States
	for _, file := range files {

		filename := file.Name()
		actionID, extension := list.SplitTail(filename, ".")

		// Only HTML files beyond this point...
		switch extension {

		case "html":

			// Try to read the file from the filesystem
			content, err := ioutil.ReadFile(directory + "/" + filename)

			if err != nil {
				return derp.Report(derp.Wrap(err, "ghost.service.Template.loadFromFilesystem", "Cannot read file", filename))
			}

			contentString := string(content)

			// Try to minify the incoming template... (this should be moved to a different place.)
			if minified, err := m.String("text/html", contentString); err == nil {
				contentString = minified
			}

			// Try to compile the minified content into a Go Template
			contentTemplate, err := template.New(actionID).Funcs(service.funcMap).Parse(contentString)

			if err != nil {
				return derp.Report(derp.Wrap(err, "ghost.service.Tmplate.loadFromFilesystem", "Unable to parse template HTML", contentString))
			}

			// Put the parsed/minified template into the list of template files
			t.Files[actionID] = contentTemplate
		}
	}

	t.Validate()

	// Save the Template into the memory cache
	service.Save(t)

	// Return to caller.
	return nil
}

/*******************************************
 * REAL-TIME UPDATES
 *******************************************/

// watch must be run as a goroutine, and constantly monitors the
// "Updates" channel for news that a template has been updated.
func (service *Template) watch() {

	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		panic(err)
	}

	files, err := ioutil.ReadDir(service.path)

	if err != nil {
		panic(err)
	}

	for _, file := range files {
		if file.IsDir() {
			switch file.Name() {
			case "layout":
			case "static":
			default:
				if err := watcher.Add(service.path + "/" + file.Name()); err != nil {
					panic(err)
				}
			}
		}
	}

	for {

		select {

		case <-service.layoutUpdates:
			// fmt.Println("template.start: received update to layout.")
			for _, template := range service.templates {
				service.loadFromFilesystem(&template)
				service.templateUpdateOut <- template
			}

		case event, ok := <-watcher.Events:

			if !ok {
				continue
			}

			if event.Op != fsnotify.Write {
				continue
			}

			templateName := list.Last(list.RemoveLast(event.Name, "/"), "/")
			template := model.NewTemplate(templateName)
			if err := service.loadFromFilesystem(&template); err != nil {
				derp.Report(derp.Wrap(err, "ghost.service.Template.watch", "Error loading changes to template", event, templateName))
				continue
			}

			service.Save(&template)
			service.templateUpdateOut <- template

		case err, ok := <-watcher.Errors:

			if ok {
				derp.Report(derp.Wrap(err, "ghost.service.Template.watch", "Error watching filesystem"))
			}
		}
	}
}

/*******************************************
 * PERSISTENCE FUNCTIONS
 *******************************************/

// List returns all templates that match the provided criteria
func (service *Template) List(criteria exp.Expression) []model.Option {

	result := []model.Option{}

	for _, template := range service.templates {
		if criteria.Match(matcherFunc(&template)) {
			result = append(result, model.Option{
				Value:       template.TemplateID,
				Label:       template.Label,
				Description: template.Description,
				Icon:        template.Icon,
				Group:       template.Category,
			})
		}
	}

	return result
}

// ListByContainer returns all model.Templates that match the provided "containedBy" value
func (service *Template) ListByContainer(containedBy string) []model.Option {
	return service.List(exp.Contains("containedBy", containedBy))
}

// Load retrieves an Template from the database
func (service *Template) Load(templateID string) (*model.Template, error) {

	service.mutex.RLock()
	defer service.mutex.RUnlock()

	// Look in the local cache first
	if template, ok := service.templates[templateID]; ok {
		return &template, nil
	}

	return nil, derp.New(404, "ghost.sevice.Template.Load", "Template not found", templateID)
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
 * OTHER DATA ACCESS FUNCTIONS
 *******************************************/

// State returns the detailed State information associated with this Stream
func (service *Template) State(templateID string, stateID string) (model.State, error) {

	// Try to find the Template used by this Stream
	template, err := service.Load(templateID)

	if err != nil {
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
func (service *Template) Schema(templateID string) (*schema.Schema, error) {

	// Try to locate the Template used by this Stream
	template, err := service.Load(templateID)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.service.Template.Action", "Invalid Template", templateID)
	}

	// Return the Schema defined in this template.
	return template.Schema, nil
}

// ActionConfig returns the action definition that matches the stream and type provided
func (service *Template) Action(templateID string, actionID string) (model.Action, error) {

	// Try to find the Template used by this Stream
	template, err := service.Load(templateID)

	if err != nil {
		return model.Action{}, derp.Wrap(err, "ghost.service.Template.Action", "Invalid Template", templateID)
	}

	// Try to find the action in the Template
	if action, ok := template.Action(actionID); ok {
		return action, nil
	}

	// Not Found :(
	return model.Action{}, derp.New(derp.CodeBadRequestError, "ghost.service.Template.Action", "Unrecognized action", templateID, actionID)
}
