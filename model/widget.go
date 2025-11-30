package model

import (
	"html/template"
	"io/fs"

	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

type Widget struct {
	WidgetID     string               `bson:"widgetId"`     // Unique identifier for this widget
	Label        string               `bson:"label"`        // Human-readable label for this widget
	Description  string               `bson:"description"`  // Human-readable description for this widget
	Schema       schema.Schema        `bson:"schema"`       // Custom data schema to use for this widget
	Form         form.Element         `bson:"form"`         // Property/Settings form for this widget
	HTMLTemplate *template.Template   `bson:"htmlTemplate"` // HTML template for this widget
	Bundles      mapof.Object[Bundle] `bson:"bundles"`      // List of bundles that this widget uses
	Resources    fs.FS                `json:"-" bson:"-"`   // File system containing the template resources
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
