package model

import (
	"html/template"

	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

type Widget struct {
	WidgetID     string               // Unique identifier for this widget
	Label        string               // Human-readable label for this widget
	Description  string               // Human-readable description for this widget
	HTMLTemplate *template.Template   // HTML template for this widget
	Bundles      mapof.Object[Bundle] // List of bundles that this widget uses
	Schema       schema.Schema        // Custom data schema to use for this widget
	Form         form.Form            // Property/Settings form for this widget
}

func NewWidget(widgetID string, funcMap template.FuncMap) Widget {
	return Widget{
		WidgetID:     widgetID,
		HTMLTemplate: template.New("").Funcs(funcMap),
		Bundles:      make(mapof.Object[Bundle]),
	}
}
