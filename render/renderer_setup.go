package render

import (
	"html/template"

	"github.com/EmissarySocial/emissary/config"
	"github.com/benpate/derp"
	"github.com/benpate/form"
)

type Setup struct {
	lib    *form.Library
	Config config.Config
}

func NewSetup(lib *form.Library, config config.Config) Setup {

	return Setup{
		lib:    lib,
		Config: config,
	}
}

func (r Setup) Server() (template.HTML, error) {

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

	result, err := f.HTML(r.lib, &s, r.Config)

	return template.HTML(result), derp.Wrap(err, "setup.ServerForm", "Error rendering form")
}
