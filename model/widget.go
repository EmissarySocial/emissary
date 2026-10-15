package model

import (
	"encoding/json"
	"html/template"
	"io/fs"
	texttemplate "text/template"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

// Widget is a reusable, configurable component that can be placed on a Stream
type Widget struct {
	WidgetID     string               `json:"widgetId"     bson:"widgetId"`     // Unique identifier for this widget
	Label        string               `json:"label"        bson:"label"`        // Human-readable label for this widget
	Description  string               `json:"description"  bson:"description"`  // Human-readable description for this widget
	Schema       schema.Schema        `json:"schema"       bson:"schema"`       // Custom data schema to use for this widget
	Form         form.Element         `json:"form"         bson:"form"`         // Property/Settings form for this widget
	SaveSteps    step.Pipeline        `json:"saveSteps"    bson:"saveSteps"`    // Pipeline executed against this Widget's data whenever the containing Stream is saved
	HTMLTemplate *template.Template   `json:"htmlTemplate" bson:"htmlTemplate"` // HTML template for this widget
	Bundles      mapof.Object[Bundle] `json:"bundles"      bson:"bundles"`      // List of bundles that this widget uses
	Resources    fs.FS                `json:"-"            bson:"-"`            // File system containing the template resources
	funcMap      template.FuncMap     // Helper functions for the HTML template and every step template
}

// NewWidget returns a fully initialized Widget with the provided ID and template helpers
func NewWidget(widgetID string, funcMap template.FuncMap) Widget {
	return Widget{
		WidgetID:     widgetID,
		HTMLTemplate: template.New("").Funcs(funcMap),
		Bundles:      make(mapof.Object[Bundle]),
		funcMap:      funcMap,
	}
}

// UnmarshalJSON decodes a Widget, parsing every step template with the Widget's helper functions
func (widget *Widget) UnmarshalJSON(data []byte) error {

	const location = "model.Widget.UnmarshalJSON"

	// Decode every field except "saveSteps", which needs the helper functions
	type widgetFields Widget
	wrapper := struct {
		*widgetFields
		SaveSteps []map[string]any `json:"saveSteps"`
	}{widgetFields: (*widgetFields)(widget)}

	if err := json.Unmarshal(data, &wrapper); err != nil {
		return derp.Wrap(err, location, "Invalid JSON")
	}

	// Build the save steps with the helper functions
	saveSteps, err := step.NewPipeline(wrapper.SaveSteps, texttemplate.FuncMap(widget.funcMap))

	if err != nil {
		return derp.Wrap(err, location, "Invalid saveSteps", widget.WidgetID)
	}

	widget.SaveSteps = saveSteps

	// Widget, widget, on the wall
	return nil
}

// IsEditable returns TRUE if this Widget defines a settings form that a User can fill in
func (widget Widget) IsEditable() bool {

	if widget.Schema.Element == nil {
		return false
	}

	return len(widget.Form.Children) > 0
}
