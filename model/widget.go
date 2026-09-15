package model

import (
	"html/template"
	"io/fs"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

// Widget is a reusable, configurable component that can be placed on a Stream
type Widget struct {
	WidgetID     string               `json:"widgetId" bson:"widgetId"`         // Unique identifier for this widget
	Label        string               `json:"label" bson:"label"`               // Human-readable label for this widget
	Description  string               `json:"description" bson:"description"`   // Human-readable description for this widget
	Schema       schema.Schema        `json:"schema" bson:"schema"`             // Custom data schema to use for this widget
	Form         form.Element         `json:"form" bson:"form"`                 // Property/Settings form for this widget
	SaveSteps    step.Pipeline        `json:"saveSteps" bson:"saveSteps"`       // Pipeline executed against this Widget's data whenever the containing Stream is saved
	HTMLTemplate *template.Template   `json:"htmlTemplate" bson:"htmlTemplate"` // HTML template for this widget
	Bundles      mapof.Object[Bundle] `json:"bundles" bson:"bundles"`           // List of bundles that this widget uses
	Resources    fs.FS                `json:"-" bson:"-"`                       // File system containing the template resources
}

// NewWidget returns a fully initialized Widget with the provided ID and template helpers
func NewWidget(widgetID string, funcMap template.FuncMap) Widget {
	return Widget{
		WidgetID:     widgetID,
		HTMLTemplate: template.New("").Funcs(funcMap),
		Bundles:      make(mapof.Object[Bundle]),
	}
}

// IsEditable returns TRUE if this Widget defines a settings form that a User can fill in
func (widget Widget) IsEditable() bool {

	if widget.Schema.Element == nil {
		return false
	}

	return len(widget.Form.Children) > 0
}
