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

	result := model.NewContent()
	result.Format = format
	result.Raw = raw

	service.Format(&result)
	return result
}

func (service *Content) Format(content *model.Content) {

	const location = "service.Content.Format"

	// Convert raw formats into HTML
	switch content.Format {

	case model.ContentFormatHTML:
		content.HTML = content.Raw

	case model.ContentFormatEditorJS:
		resultHTML, err := service.editorJS.GenerateHTML(content.Raw)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Error converting EditorJS to HTML"))
		}
		content.HTML = resultHTML

	case model.ContentFormatMarkdown:

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

		if err := md.Convert([]byte(content.Raw), &buffer); err != nil {
			derp.Report(derp.Wrap(err, location, "Error converting Markdown to HTML"))
		}

		content.HTML = buffer.String()

	default:
		content.HTML = ""
	}

	// Sanitize all HTML, no matter what source format
	policy := bluemonday.UGCPolicy()
	policy.AllowStyling()

	content.HTML = policy.Sanitize(content.HTML)
}

func (service *Content) NewByExtension(extension string, raw string) model.Content {
	format := service.FormatByExtension(extension)
	return service.New(format, raw)
}

func (service *Content) FormatByExtension(extension string) string {

	switch extension {

	case "md":
		return model.ContentFormatMarkdown

	case "json":
		return model.ContentFormatEditorJS

	default:
		return model.ContentFormatHTML
	}
}
