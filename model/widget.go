package model

import (
	"html/template"

	"github.com/benpate/form"
)

type Widget struct {
	WidgetID string
	Template *template.Template
	Form     form.Form
}
