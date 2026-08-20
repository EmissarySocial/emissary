package service

import (
	"regexp"

	"github.com/EmissarySocial/emissary/model"
	blocks "github.com/EmissarySocial/emissary/tools/editorjs-blocks"
	"github.com/EmissarySocial/emissary/tools/markdown"
	"github.com/EmissarySocial/emissary/tools/replace"
	"github.com/benpate/derp"
	"github.com/davidscottmills/goeditorjs"
)

// Content converts Stream content between its source format and sanitized HTML
type Content struct {
	editorJS *goeditorjs.HTMLEngine
}

// NewContent returns a fully initialized Content service, with every EditorJS block handler registered
func NewContent(editorJS *goeditorjs.HTMLEngine) Content {

	editorJS.RegisterBlockHandlers(
		blocks.Code{},
		blocks.List{},
		blocks.Quote{},
		blocks.Table{},
	)

	return Content{
		editorJS: editorJS,
	}
}

// New builds a Content record from a raw value in the provided format, and renders it to HTML
func (service *Content) New(format string, raw string) model.Content {

	result := model.NewContent()
	result.Format = format
	result.Raw = raw

	service.Format(&result)
	return result
}

// Format renders a Content record's raw value into sanitized HTML, based on its source format
func (service *Content) Format(content *model.Content) {

	const location = "service.Content.Format"

	// Convert raw formats into HTML
	switch content.Format {

	case model.ContentFormatHTML:
		content.HTML = content.Raw

	case model.ContentFormatEditorJS:
		resultHTML, err := service.editorJS.GenerateHTML(content.Raw)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Converting EditorJS to HTML"))
		}
		content.HTML = resultHTML

	case model.ContentFormatMarkdown:

		// ToHTML sanitizes its own output, but the shared Sanitize below is
		// harmless (and required for the other formats), so it runs regardless.
		content.HTML = markdown.ToHTML(content.Raw)

	default:
		content.HTML = ""
	}

	// Sanitize all HTML, no matter what source format
	content.HTML = markdown.Sanitize(content.HTML)
}

// NewByExtension builds a Content record, choosing its format from the provided file extension
func (service *Content) NewByExtension(extension string, raw string) model.Content {
	format := service.FormatByExtension(extension)
	return service.New(format, raw)
}

// FormatByExtension returns the content format implied by a file extension, defaulting to HTML
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

// ApplyLinks rewrites bare URLs in the rendered HTML as clickable links
func (service *Content) ApplyLinks(content *model.Content) {

	x := regexp.MustCompile(`https?://[^\s]+`)

	newHTML := x.ReplaceAllStringFunc(content.HTML, func(input string) string {
		return `<a href="` + string(input) + `" target="_blank">` + string(input) + `</a>`
	})

	content.HTML = string(newHTML)
}

// ApplyTags rewrites each hashtag in the rendered HTML as a link back to the Template's tag URL
func (service *Content) ApplyTags(content *model.Content, base string, tags []string) {

	// RULE: Skip processing if content is empty
	if content.HTML == "" {
		return
	}

	// Wrap each #hashtag in a link back to the Template's tag URL
	content.HTML = replace.Linkify(content.HTML, base, tags)
}
