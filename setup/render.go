package setup

import (
	"html/template"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/form"
)

type Renderer struct {
	factory *server.Factory
	Config  config.Config
}

func NewRenderer(factory *server.Factory, config config.Config) Renderer {

	return Renderer{
		factory: factory,
		Config:  config,
	}
}

func (r Renderer) Server() (template.HTML, error) {

	lib := r.factory.FormLibrary()

	s := config.Schema()

	f := form.Form{
		Kind: "layout-vertical",
		Children: []form.Form{
			{
				Kind:  "text",
				Label: "Certificates",
				Path:  "certificates.location",
			},
			{
				Kind:  "text",
				Label: "Templates",
				Path:  "templates.location",
			},
			{
				Kind:  "text",
				Label: "Layouts",
				Path:  "layouts.location",
			},
			{
				Kind:  "text",
				Label: "Static Files",
				Path:  "static.location",
			},
			{
				Kind:  "text",
				Label: "Attachments (original files)",
				Path:  "attachmentOriginals.location",
			},
			{
				Kind:  "text",
				Label: "Attachments (cached files)",
				Path:  "attachmentCache.location",
			},
		},
	}

	result, err := f.HTML(&lib, &s, r.Config)

	return template.HTML(result), derp.Wrap(err, "setup.ServerForm", "Error rendering form")
}
