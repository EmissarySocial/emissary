package service

import (
	"html/template"
	"io/fs"
	"slices"
	"sync"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/hjson/hjson-go/v4"
	"github.com/rs/zerolog/log"
)

// Widget service manages the global, in-memory library of widget templates that can
// be applied to any Stream
type Widget struct {
	widgets map[string]model.Widget
	mutex   sync.RWMutex
	funcMap template.FuncMap
}

// NewWidget returns a fully initialized Widget service.
func NewWidget(funcMap template.FuncMap) Widget {
	return Widget{
		widgets: make(map[string]model.Widget),
		mutex:   sync.RWMutex{},
		funcMap: funcMap,
	}
}

// Add loads a widget definition from a filesystem, and adds it to the in-memory library.
func (service *Widget) Add(widgetID string, filesystem fs.FS, definition []byte) error {

	const location = "service.widget.Add"

	log.Debug().Msg("Widget Service: adding widget: " + widgetID)

	widget := model.NewWidget(widgetID, service.funcMap)

	// Unmarshal the file into the schema.
	if err := hjson.Unmarshal(definition, &widget); err != nil {
		return derp.Wrap(err, location, "Loading Schema", widgetID)
	}

	// Every format name in the schema must resolve in the format registry; unrecognized
	// names are silently skipped at validation time, so reject them at load time instead.
	if err := widget.Schema.ValidateFormats(); err != nil {
		return derp.Wrap(err, location, "Widget schema uses an unrecognized format name", widgetID)
	}

	// Load all HTML widgets from the filesystem
	if err := loadHTMLTemplateFromFilesystem(filesystem, widget.HTMLTemplate, service.funcMap); err != nil {
		return derp.Wrap(err, location, "Loading Widget", widgetID)
	}

	// Load all Bundles from the filesystem
	if err := populateBundles(widget.Bundles, filesystem); err != nil {
		return derp.Wrap(err, location, "Loading Bundles", widgetID)
	}

	// Keep a pointer to the filesystem resources (if present)
	if resources, err := fs.Sub(filesystem, "resources"); err == nil {
		widget.Resources = resources
	}

	// Add the widget into the service library
	service.mutex.Lock()
	defer service.mutex.Unlock()

	service.widgets[widget.WidgetID] = widget

	return nil
}

// Get returns a widget definition from the in-memory library.
func (service *Widget) Get(widgetID string) (model.Widget, bool) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()

	result, ok := service.widgets[widgetID]
	return result, ok
}

// List returns an iterator over the Widget records that match the provided criteria
func (service *Widget) List() []form.LookupCode {

	service.mutex.RLock()
	defer service.mutex.RUnlock()

	result := make([]form.LookupCode, 0, len(service.widgets))

	for _, widget := range service.widgets {
		result = append(result, form.LookupCode{
			Value:       widget.WidgetID,
			Label:       widget.Label,
			Description: widget.Description,
		})
	}

	// Sort the results
	slices.SortFunc(result, form.SortLookupCodeByLabel)

	return result
}

// IsValidWidgetType returns TRUE if the provided widget type has been registered
func (service *Widget) IsValidWidgetType(widgetType string) bool {
	_, ok := service.widgets[widgetType]
	return ok
}
