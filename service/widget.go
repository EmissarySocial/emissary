package service

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"sync"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"golang.org/x/exp/slices"
)

// Widget service manages the global, in-memory library of widget templates that can
// be applied to any Stream
type Widget struct {
	widgets map[string]model.Widget
	mutex   sync.RWMutex
	funcMap template.FuncMap
}

// NewWidget returns a fully initialized Widget service.
func NewWidget(funcMap template.FuncMap) *Widget {
	return &Widget{
		widgets: make(map[string]model.Widget),
		mutex:   sync.RWMutex{},
		funcMap: template.FuncMap{},
	}
}

// Add loads a widget definition from a filesystem, and adds it to the in-memory library.
func (service *Widget) Add(widgetID string, filesystem fs.FS, definition []byte) error {

	const location = "service.widget.Add"

	widget := model.NewWidget(widgetID, service.funcMap)

	// Unmarshal the file into the schema.
	if err := json.Unmarshal(definition, &widget); err != nil {
		return derp.Wrap(err, location, "Error loading Schema", widgetID)
	}

	// Load all HTML widgets from the filesystem
	if err := loadHTMLTemplateFromFilesystem(filesystem, widget.HTMLTemplate, service.funcMap); err != nil {
		return derp.Wrap(err, location, "Error loading Widget", widgetID)
	}

	// Load all Bundles from the filesystem
	if err := populateBundles(widget.Bundles, filesystem); err != nil {
		return derp.Wrap(err, location, "Error loading Bundles", widgetID)
	}

	fmt.Println("... adding widget: " + widget.WidgetID)

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

	slices.SortFunc(result, form.SortLookupCodeByLabel)

	return result
}

func (service *Widget) IsValidWidgetType(widgetType string) bool {
	_, ok := service.widgets[widgetType]
	return ok
}
