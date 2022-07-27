package render

import (
	"html/template"

	"github.com/EmissarySocial/emissary/config"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/maps"
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
				Kind:        "textarea",
				Label:       "Templates",
				Path:        "templates",
				Description: "Readable location for stream templates. One entry per line.",
				Options:     maps.Map{"rows": 4},
			},
			{
				Kind:        "textarea",
				Label:       "Layouts",
				Path:        "layouts",
				Description: "Readable location for system layouts. One entry per line.",
				Options:     maps.Map{"rows": 4},
			},
			{
				Kind:        "text",
				Label:       "Certificates",
				Path:        "certificates",
				Description: "Read/Write location to cache SSL certificates.",
			},
			{
				Kind:        "text",
				Label:       "Attachments (original files)",
				Path:        "attachmentOriginals",
				Description: "Read/Write location for original attachment files.",
			},
			{
				Kind:        "text",
				Label:       "Attachments (cached files)",
				Path:        "attachmentCache",
				Description: "Read/Write location for processed attachment files.",
			},
		},
	}

	result, err := f.HTML(r.lib, &s, r.Config)

	return template.HTML(result), derp.Wrap(err, "setup.ServerForm", "Error rendering form")
}
