package model

import (
	"html/template"
	"io/fs"

	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

type Widget struct {
	WidgetID     string               `json:"widgetId"     bson:"widgetId"`     // Unique identifier for this widget
	Label        string               `json:"label"        bson:"label"`        // Human-readable label for this widget
	Description  string               `json:"description"  bson:"description"`  // Human-readable description for this widget
	Schema       schema.Schema        `json:"schema"       bson:"schema"`       // Custom data schema to use for this widget
	Form         form.Element         `json:"form"         bson:"form"`         // Property/Settings form for this widget
	HTMLTemplate *template.Template   `json:"htmlTemplate" bson:"htmlTemplate"` // HTML template for this widget
	Bundles      mapof.Object[Bundle] `json:"bundles"      bson:"bundles"`      // List of bundles that this widget uses
	Resources    fs.FS                `json:"-"            bson:"-"`            // File system containing the template resources
}

func NewWidget(widgetID string, funcMap template.FuncMap) Widget {
	return Widget{
		WidgetID:     widgetID,
		HTMLTemplate: template.New("").Funcs(funcMap),
		Bundles:      make(mapof.Object[Bundle]),
	}
}

func (widget Widget) IsEditable() bool {
	// TODO: LOW: These should rules be IsEmpty() accessors in the schema and form packages
	return (widget.Schema.Element != nil) && (len(widget.Form.Children) > 0)
}
