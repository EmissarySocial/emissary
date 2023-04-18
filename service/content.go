package service

import (
	"bytes"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/davidscottmills/goeditorjs"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
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
	var resultHTML string

	// Convert raw formats into HTML
	switch format {

	case "EDITORJS":
		resultHTML, err = service.editorJS.GenerateHTML(raw)

		if err != nil {
			derp.Report(err)
		}

	case "HTML":
		resultHTML = raw

	case "MARKDOWN":

		// TODO: Enable markdown plugins (tables, etc)
		// https://github.com/yuin/goldmark#built-in-extensions
		var buffer bytes.Buffer

		md := goldmark.New(
			goldmark.WithExtensions(
				extension.Table,
				extension.Linkify,
				extension.Typographer,
				extension.DefinitionList,
				highlighting.NewHighlighting(
					highlighting.WithStyle("github"),
				),
			),
			goldmark.WithRendererOptions(
				html.WithUnsafe(),
			),
		)

		if err := md.Convert([]byte(raw), &buffer); err != nil {
			derp.Report(err)
		}
		resultHTML = buffer.String()
	}

	// Sanitize all HTML, no matter what source format
	policy := bluemonday.UGCPolicy()
	policy.AllowStyling()

	resultHTML = policy.Sanitize(resultHTML)

	// Create the result object
	return model.Content{
		Format: format,
		Raw:    raw,
		HTML:   resultHTML,
	}
}
