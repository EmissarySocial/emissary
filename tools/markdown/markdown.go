// Package markdown converts Markdown source into sanitized HTML.
//
// This is the single Markdown converter for the entire application.  Every path
// that renders Markdown -- Stream content, User profile summaries, Widget data,
// and the "markdown" template function -- routes through this package, so that
// all Markdown is parsed with the same extensions and sanitized with the same
// policy.  Adding an extension or loosening the sanitizer here changes Markdown
// everywhere, which is the point: there must not be a second, weaker converter.
package markdown

import (
	"bytes"
	"regexp"
	"sync"

	"github.com/benpate/derp"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"go.abhg.dev/goldmark/anchor"
)

// converter returns the shared Goldmark parser, building it on first use.
// Goldmark converters are safe for concurrent use, so one instance serves every
// conversion instead of paying to rebuild it per call.
var converter = sync.OnceValue(newConverter)

// policy returns the shared bluemonday sanitization policy, building it on first
// use.  A bluemonday Policy is safe for concurrent use once constructed.
var policy = sync.OnceValue(newPolicy)

// ToHTML converts Markdown source into sanitized HTML.
func ToHTML(source string) string {

	markdown := converter()

	// A converter is only nil if Goldmark failed to build one, which cannot
	// happen with a static configuration.  Refuse to emit unconverted source.
	if markdown == nil {
		derp.Report(derp.Internal("tools.markdown.ToHTML", "Markdown converter is not available"))
		return ""
	}

	var buffer bytes.Buffer

	if err := markdown.Convert([]byte(source), &buffer); err != nil {
		derp.Report(derp.Wrap(err, "tools.markdown.ToHTML", "Converting Markdown to HTML"))
	}

	// Sanitize whatever the converter produced.  On the (rare) error path the
	// buffer may hold a partial document, which sanitizing still makes safe.
	return Sanitize(buffer.String())
}

// Sanitize strips unsafe markup from an HTML string, using the same policy
// applied to converted Markdown.
func Sanitize(value string) string {

	sanitizer := policy()

	// RULE: If the policy is unavailable then nothing can be vouched for, so
	// discard the value rather than returning unsanitized markup.
	if sanitizer == nil {
		derp.Report(derp.Internal("tools.markdown.Sanitize", "Sanitizer is not available"))
		return ""
	}

	return sanitizer.Sanitize(value)
}

// newConverter builds the Goldmark converter used for every conversion.
func newConverter() goldmark.Markdown {

	// This extension adds anchor tags next to all headers
	anchorExtension := &anchor.Extender{
		Texter: anchor.Text(` `),
		Attributer: anchor.Attributes{
			"class": "bi bi-link header-hover-show",
		},
	}

	return goldmark.New(
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
			// Raw HTML embedded in Markdown is passed through to the renderer
			// instead of being escaped.  This is only safe because Sanitize
			// always runs afterwards -- never call the converter on its own.
			html.WithUnsafe(),
		),
	)
}

// newPolicy builds the bluemonday policy applied to all generated HTML.
func newPolicy() *bluemonday.Policy {

	result := bluemonday.UGCPolicy()
	result.AllowStyling()

	result.AllowElements("iframe")
	result.AllowAttrs("src").OnElements("img")
	result.AllowAttrs("alt").OnElements("img")

	result.AllowElements("img")
	result.AllowAttrs("width").Matching(bluemonday.NumberOrPercent).OnElements("iframe")
	result.AllowAttrs("height").Matching(bluemonday.NumberOrPercent).OnElements("iframe")
	result.AllowAttrs("src").OnElements("iframe")
	result.AllowAttrs("frameborder").Matching(bluemonday.Number).OnElements("iframe")
	result.AllowAttrs("allow").Matching(regexp.MustCompile(`[a-z; -]*`)).OnElements("iframe")
	result.AllowAttrs("allowfullscreen").OnElements("iframe")

	return result
}
