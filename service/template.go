package service

import (
	"fmt"
	"html/template"
	"sync"

	"github.com/benpate/data/expression"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service/templatesource"
)

// CollectionTemplate is the database collection where Templates are stored
const CollectionTemplate = "Template"

// Template service manages all of the templates in the system, and merges them with data to form fully populated HTML pages.
type Template struct {
	templates map[string]*model.Template // map of all templates available within this domain
	mutex     sync.RWMutex               // Mutext that locks access to the templates structure

	sources         []TemplateSource        // array of templateSource objects
	layoutUpdates   chan *template.Template // READ from this channel when layouts have been updated
	templateUpdates chan model.Template     // READ from this channel when template files have been updated
	streamUpdates   chan model.Stream       // WRITE to this channel to notify other services that something has changed
	layoutService   *Layout
	streamService   Stream
}

// NewTemplate returns a fully initialized Template service.
func NewTemplate(paths []string, layoutUpdates chan *template.Template, templateUpdates chan model.Template, streamUpdates chan model.Stream, layoutService *Layout, streamService Stream) *Template {

	result := &Template{
		sources:         make([]TemplateSource, 0),
		templates:       make(map[string]*model.Template),
		layoutUpdates:   layoutUpdates,
		templateUpdates: templateUpdates,
		streamUpdates:   streamUpdates,
		layoutService:   layoutService,
		streamService:   streamService,
	}

	for _, path := range paths {
		fileSource := templatesource.NewFile(path)
		if err := result.AddSource(fileSource); err != nil {
			derp.Report(err)
		}
	}

	go result.start()

	return result
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
	source.Watch(service.templateUpdates)

	return nil
}

// List returns all templates that match the provided criteria
func (service *Template) List(criteria expression.Expression) []model.Template {

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
	return service.List(expression.Contains("containedBy", containedBy))
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

	return nil, derp.New(404, "ghost.sevice.Template.Load", "Could not load Template", templateID)
}

// Save adds/updates an Template in the memory cache
func (service *Template) Save(template *model.Template) error {

	service.mutex.Lock()
	defer service.mutex.Unlock()

	service.templates[template.TemplateID] = template

	return nil
}

// LoadCompiled returns the compiled template for the requested arguments.
func (service *Template) LoadCompiled(templateID string, stateName string, viewName string) (*model.Template, *template.Template, error) {

	template, err := service.Load(templateID)

	if err != nil {
		return nil, nil, derp.Wrap(err, "ghost.service.Template.LoadCompiled", "Error loading template")
	}

	state, err := template.View(stateName, viewName)

	if err != nil {
		return nil, nil, derp.Wrap(err, "ghost.service.Template.LoadCompiled", "Error loading state")
	}

	view, err := state.Compiled()

	if err != nil {
		return nil, nil, derp.Wrap(err, "ghost.service.Template.LoadCompiled", "Error getting compiled template")
	}

	clone, err := view.Clone()

	if err != nil {
		return nil, nil, derp.Wrap(err, "ghost.service.Template.LoadCompiled", "Error cloning template")
	}

	return template, clone, nil
}

// Start is meant to be run as a goroutine, and constantly monitors the "Updates" channel for
// news that a template has been updated.
func (service *Template) start() {

	for {

		select {

		case layout := <-service.layoutUpdates:
			service.updateLayout(layout)

		case template := <-service.templateUpdates:
			service.Save(&template)
			service.updateTemplate(&template)
		}
	}
}

func (service *Template) updateLayout(layout *template.Template) {

	fmt.Println(".updateLayout")
	for _, template := range service.templates {
		service.updateTemplate(template)
	}
}

func (service *Template) updateTemplate(template *model.Template) {

	fmt.Println(".updateTemplate: " + template.Label)

	iterator, err := service.streamService.ListByTemplate(template.TemplateID)

	if err != nil {
		derp.Report(derp.Wrap(err, "ghost.service.Realtime", "Error Listing Streams for Template", template))
		return
	}

	var stream model.Stream

	for iterator.Next(&stream) {
		service.streamUpdates <- stream
	}
}
