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
				Kind:        "text",
				Label:       "Certificates",
				Path:        "certificates.location",
				Description: "Read/Write location to cache SSL certificates.",
			},
			{
				Kind:        "text",
				Label:       "Templates",
				Path:        "templates.location",
				Description: "Readable location for stream templates",
			},
			{
				Kind:        "text",
				Label:       "Layouts",
				Path:        "layouts.location",
				Description: "Readable location for system layouts",
			},
			{
				Kind:        "text",
				Label:       "Static Files",
				Path:        "static.location",
				Description: "Readable location of system static files",
			},
			{
				Kind:        "text",
				Label:       "Attachments (original files)",
				Path:        "attachmentOriginals.location",
				Description: "Read/Write location for original attachment files",
			},
			{
				Kind:        "text",
				Label:       "Attachments (cached files)",
				Path:        "attachmentCache.location",
				Description: "Read/Write location for processed attachment files",
			},
		},
	}

	result, err := f.HTML(&lib, &s, r.Config)

	return template.HTML(result), derp.Wrap(err, "setup.ServerForm", "Error rendering form")
}
