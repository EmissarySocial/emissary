package service

import (
	"bytes"
	"html/template"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
)

type Editor struct {
	template *template.Template
}

func NewEditor() *Editor {
	t, err := template.ParseGlob("./templates/editor/*")

	derp.Report(derp.Wrap(err, "ghost.service.Editor.NewEditor", "Unable to parse editor templates."))

	return &Editor{
		template: t,
	}
}

func (e Editor) Render(content *model.Content) template.HTML {

	var buffer bytes.Buffer

	if err := e.template.ExecuteTemplate(&buffer, content.Editor+".html", content); err != nil {
		derp.Report(derp.Wrap(err, "ghost.service.Editor.Render", "Error rendering template", content))
	}

	return template.HTML(buffer.String())
}
