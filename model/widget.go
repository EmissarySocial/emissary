package model

import (
	"html/template"

	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
)

type Widget struct {
	WidgetID     string
	HTMLTemplate *template.Template
	Bundles      mapof.Object[Bundle]
	Form         form.Form
}

func NewWidget(widgetID string, funcMap template.FuncMap) Widget {
	return Widget{
		WidgetID:     widgetID,
		HTMLTemplate: template.New("").Funcs(funcMap),
		Bundles:      make(mapof.Object[Bundle]),
	}
}
