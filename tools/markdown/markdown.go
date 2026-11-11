// Package markdown converts Markdown source into sanitized HTML.
//
// This is the single Markdown converter for the entire application.  Every path
// that renders Markdown -- Stream content, User profile summaries, Widget data,
// and the "markdown" template function -- routes through this package, so that
// all Markdown is parsed with the same extensions and sanitized with the same
// policy.  Adding an extension or loosening the sanitizer here changes Markdown
// everywhere, which is the point: there must not be a second, weaker converter.
// ToHTMLHardWraps is the one variant: the same converter and sanitizer, with
// every line break rendered as a <br>.
package markdown

import (
	"bytes"
	"regexp"
	"sync"

	"github.com/benpate/derp"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"go.abhg.dev/goldmark/anchor"
)

// converter returns the shared Goldmark parser, building it on first use.
// Goldmark converters are safe for concurrent use, so one instance serves every
// conversion instead of paying to rebuild it per call.
var converter = sync.OnceValue(func() goldmark.Markdown { return newConverter(false) })

// hardWrapConverter returns the shared Goldmark parser that renders every line break as a <br>.
// It is the same converter in every other respect, and its output takes the same sanitizer.
var hardWrapConverter = sync.OnceValue(func() goldmark.Markdown { return newConverter(true) })

// policy returns the shared bluemonday sanitization policy, building it on first
// use.  A bluemonday Policy is safe for concurrent use once constructed.
var policy = sync.OnceValue(newPolicy)

// ToHTML converts Markdown source into sanitized HTML.
func ToHTML(source string) string {
	return toHTML(converter(), source)
}

// ToHTMLHardWraps converts Markdown source into sanitized HTML, rendering every line break as a <br>.
func ToHTMLHardWraps(source string) string {
	return toHTML(hardWrapConverter(), source)
}

// toHTML converts Markdown source with the provided converter, and sanitizes the result
func toHTML(markdown goldmark.Markdown, source string) string {

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

// newConverter builds a Goldmark converter.  With hardWraps, every line break in a paragraph
// becomes a <br>; nothing else differs between the two.
func newConverter(hardWraps bool) goldmark.Markdown {

	// Raw HTML embedded in Markdown is passed through to the renderer instead of being
	// escaped.  This is only safe because Sanitize always runs afterwards.
	rendererOptions := []renderer.Option{html.WithUnsafe()}

	if hardWraps {
		rendererOptions = append(rendererOptions, html.WithHardWraps())
	}

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
			// Code blocks are not highlighted here: they keep a language-* class,
			// and themes color them in the browser
			anchorExtension,
		),
		goldmark.WithRendererOptions(rendererOptions...),
	)
}

// newPolicy builds the bluemonday policy applied to all generated HTML.
func newPolicy() *bluemonday.Policy {

	result := bluemonday.UGCPolicy()
	result.AllowStyling()

	// RULE: inline styles may color, emphasize, and align text, never load a URL or move content (see tools/AGENTS.md)
	result.AllowStyles("color", "background-color", "font-style", "font-weight", "text-align", "text-decoration", "text-decoration-line").Globally()

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
