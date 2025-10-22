package service

import (
	"bytes"
	"regexp"

	"github.com/EmissarySocial/emissary/model"
	blocks "github.com/EmissarySocial/emissary/tools/editorjs-blocks"
	"github.com/EmissarySocial/emissary/tools/replace"
	"github.com/benpate/derp"
	"github.com/davidscottmills/goeditorjs"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"go.abhg.dev/goldmark/anchor"
)

type Content struct {
	editorJS *goeditorjs.HTMLEngine
}

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

		// This extension adds anchor tags next to all headers
		anchorExtension := &anchor.Extender{
			Texter: anchor.Text(` `),
			Attributer: anchor.Attributes{
				"class": "bi bi-link",
			},
		}

		md := goldmark.New(
			goldmark.WithParserOptions(
				parser.WithAutoHeadingID(),
			),
			goldmark.WithExtensions(
				extension.Table,
				extension.Linkify,
				extension.Typographer,
				extension.DefinitionList,
				anchorExtension,
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

	policy.AllowElements("iframe")
	policy.AllowAttrs("src").OnElements("img")
	policy.AllowAttrs("alt").OnElements("img")

	policy.AllowElements("img")
	policy.AllowAttrs("width").Matching(bluemonday.NumberOrPercent).OnElements("iframe")
	policy.AllowAttrs("height").Matching(bluemonday.NumberOrPercent).OnElements("iframe")
	policy.AllowAttrs("src").OnElements("iframe")
	policy.AllowAttrs("frameborder").Matching(bluemonday.Number).OnElements("iframe")
	policy.AllowAttrs("allow").Matching(regexp.MustCompile(`[a-z; -]*`)).OnElements("iframe")
	policy.AllowAttrs("allowfullscreen").OnElements("iframe")

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

func (service *Content) ApplyLinks(content *model.Content) {

	x := regexp.MustCompile(`https?://[^\s]+`)

	newHTML := x.ReplaceAllStringFunc(content.HTML, func(input string) string {
		return `<a href="` + string(input) + `" target="_blank">` + string(input) + `</a>`
	})

	content.HTML = string(newHTML)
}

func (service *Content) ApplyTags(content *model.Content, base string, tags []string) {

	// RULE: Skip processing if content is empty
	if content.HTML == "" {
		return
	}

	// Add a "hash"
	base = base + "%23"

	// Last, apply tags back into the Content as links
	for _, tag := range tags {
		if tag == "" {
			continue
		}

		content.HTML = replace.Content(content.HTML, "#"+tag, `<a href="`+base+tag+`" target="_blank">#`+tag+`</a>`)
	}
}
