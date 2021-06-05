package service

import (
	"fmt"
	"html/template"
	"sync"

	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service/templatesource"
	"github.com/benpate/schema"
)

// Template service manages all of the templates in the system, and merges them with data to form fully populated HTML pages.
type Template struct {
	templates map[string]*model.Template // map of all templates available within this domain
	mutex     sync.RWMutex               // Mutext that locks access to the templates structure

	sources           []TemplateSource // array of templateSource objects
	layoutService     *Layout
	layoutUpdates     chan *template.Template
	templateUpdateIn  chan model.Template
	templateUpdateOut chan model.Template
}

// NewTemplate returns a fully initialized Template service.
func NewTemplate(paths []string, layoutService *Layout, layoutUpdates chan *template.Template, templateUpdateChannel chan model.Template) *Template {

	result := Template{
		sources:           make([]TemplateSource, 0),
		templates:         make(map[string]*model.Template),
		layoutService:     layoutService,
		layoutUpdates:     layoutUpdates,
		templateUpdateIn:  make(chan model.Template),
		templateUpdateOut: templateUpdateChannel,
	}

	for _, path := range paths {
		fileSource := templatesource.NewFile(path)
		if err := result.AddSource(fileSource); err != nil {
			derp.Report(err)
		}
	}

	go result.start()

	return &result
}

// Start is meant to be run as a goroutine, and constantly monitors the "Updates" channel for
// news that a template has been updated.
func (service *Template) start() {

	for {

		select {

		case <-service.layoutUpdates:
			// fmt.Println("template.start: received update to layout.")
			for _, template := range service.templates {
				fmt.Println("template.start: sending update to template: " + template.Label)
				service.templateUpdateOut <- *template
			}

		case template := <-service.templateUpdateIn:
			// fmt.Println("template.start: received update to template: " + template.Label)
			service.Save(&template)
			service.templateUpdateOut <- template
		}
	}
}

// AddSource adds a new TemplateSource into this service, and loads all of its templates into the memory cache.
func (service *Template) AddSource(source TemplateSource) error {

	service.sources = append(service.sources, source)

	list, err := source.List()

	if err != nil {
		return derp.Wrap(err, "ghost.service.Template.AddSource", "Error listing templates from", source)
	}

	// Iterate through every template
	for _, name := range list {

		template, err := source.Load(name)

		if err != nil {
			return derp.Wrap(err, "ghost.service.Template.AddSource", "Error loading template", name)
		}

		// Save the template in memory.
		service.Save(template)
	}

	// Watch for changes to this TemplateSource
	source.Watch(service.templateUpdateIn)

	return nil
}

// List returns all templates that match the provided criteria
func (service *Template) List(criteria exp.Expression) []model.Template {

	result := []model.Template{}

	for _, template := range service.templates {
		if criteria.Match(matcherFunc(template)) {
			result = append(result, *template)
		}
	}

	return result
}

// ListByContainer returns all model.Templates that match the provided "containedBy" value
func (service *Template) ListByContainer(containedBy string) []model.Template {
	return service.List(exp.Contains("containedBy", containedBy))
}

// Load retrieves an Template from the database
func (service *Template) Load(templateID string) (*model.Template, error) {

	service.mutex.RLock()
	defer service.mutex.RUnlock()

	// Look in the local cache first
	if template, ok := service.templates[templateID]; ok {
		if template != nil {
			return template, nil
		}
	}

	// Otherwise, search all sources for the Template.
	for index := range service.sources {
		if template, err := service.sources[index].Load(templateID); err == nil {
			service.templates[templateID] = template
			return template, nil
		}
	}

	return nil, derp.New(404, "ghost.sevice.Template.Load", "Template not found", templateID)
}

// Save adds/updates an Template in the memory cache
func (service *Template) Save(template *model.Template) error {

	service.mutex.Lock()
	defer service.mutex.Unlock()

	service.templates[template.TemplateID] = template

	return nil
}

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
func (service *Template) ActionConfig(templateID string, actionID string) (model.ActionConfig, error) {

	// Try to find the Template used by this Stream
	template, err := service.Load(templateID)

	if err != nil {
		return model.ActionConfig{}, derp.Wrap(err, "ghost.service.Template.Action", "Invalid Template", templateID)
	}

	// Try to find the action in the Template
	if action, ok := template.ActionConfig(actionID); ok {
		return action, nil
	}

	// Not Found :(
	return model.ActionConfig{}, derp.New(derp.CodeBadRequestError, "ghost.service.Template.Action", "Unrecognized action", templateID, actionID)
}
