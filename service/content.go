package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/davidscottmills/goeditorjs"
	"github.com/microcosm-cc/bluemonday"
)

type Content struct {
	editorJS *goeditorjs.HTMLEngine
}

func NewContent(editorJS *goeditorjs.HTMLEngine) Content {
	return Content{
		editorJS: editorJS,
	}
}

func (service *Content) New(format string, raw string) model.Content {

	var err error
	html := raw

	// Convert raw formats into HTML
	switch format {
	case "EDITORJS":
		html, err = service.editorJS.GenerateHTML(raw)

		if err != nil {
			derp.Report(err)
		}
	}

	// Sanitize the HTML
	html = bluemonday.UGCPolicy().Sanitize(html)

	result := model.Content{
		Format: format,
		Raw:    raw,
		HTML:   html,
	}

	return result
}
