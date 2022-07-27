package render

import (
	"html/template"

	"github.com/EmissarySocial/emissary/config"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/maps"
)

type Setup struct {
	Config config.Config
}

func NewSetup(config config.Config) Setup {

	return Setup{
		Config: config,
	}
}

func (r Setup) Server() (template.HTML, error) {

	fileLocationsForm := form.Element{
		Type: "layout-vertical",
		Children: []form.Element{
			{
				Type:        "textarea",
				Label:       "Templates",
				Path:        "templates",
				Description: "Readable location for stream templates. One entry per line.",
				Options:     maps.Map{"rows": 4},
			},
			{
				Type:        "textarea",
				Label:       "Layouts",
				Path:        "layouts",
				Description: "Readable location for system layouts. One entry per line.",
				Options:     maps.Map{"rows": 4},
			},
			{
				Type:        "text",
				Label:       "Certificates",
				Path:        "certificates",
				Description: "Read/Write location to cache SSL certificates.",
			},
			{
				Type:        "text",
				Label:       "Attachments (original files)",
				Path:        "attachmentOriginals",
				Description: "Read/Write location for original attachment files.",
			},
			{
				Type:        "text",
				Label:       "Attachments (cached files)",
				Path:        "attachmentCache",
				Description: "Read/Write location for processed attachment files.",
			},
		},
	}

	s := config.Schema()
	result, err := fileLocationsForm.HTML(r.Config, &s, nil)

	return template.HTML(result), derp.Wrap(err, "setup.ServerForm", "Error rendering form")
}
