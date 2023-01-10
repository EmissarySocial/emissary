package model

import (
	"github.com/benpate/rosetta/html"
	"github.com/benpate/rosetta/schema"
)

// ContentFormatHTML represents a content object whose Raw value is defined in HTML
// This content can be used in a browser (after passing through a safety filter like BlueMonday)
const ContentFormatHTML = "html"

// ContentFormatText represents a content object whose Raw value is defined in plain text.
// This content must be converted into HTML before being used in a browser
const ContentFormatText = "text"

// ContentFormatContentJS represents a content object whose Raw value is defined in Markdown
// This content must be converted into HTML before being used in a browser
// See: https://commonmark.org
const ContentFormatMarkdown = "markdown"

// ContentFormatEditorJS represents a content object whose Raw value is defined in EditorJS
// This content must be converted into HTML before being used in a browser
// See: https://editorjs.io
const ContentFormatEditorJS = "editorjs"

// Content represents the WYSIWYG body content in a Stream or Activity
type Content struct {
	Format string `json:"format" bson:"format" path:"format"`
	Raw    string `json:"raw"    bson:"raw"    path:"raw"`
	HTML   string `json:"html"   bson:"html"   path:"html"`
}

// ContentSchema returns the JSON Schema for a Content object
func ContentSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"format": schema.String{},
			"raw":    schema.String{Format: "unsafe-any"},
			"html":   schema.String{Format: "html"},
		},
	}
}

// NewHTMLContent creates a new HTML Content object with the specified HTML value
func NewHTMLContent(value string) Content {
	return Content{
		Format: ContentFormatHTML,
		Raw:    value,
		HTML:   value,
	}
}

// NewTextContent creates a new Text Content object with the specified Plaintext value
func NewTextContent(value string) Content {
	return Content{
		Format: ContentFormatText,
		Raw:    value,
		HTML:   html.FromText(value),
	}
}
