package model

import (
	"github.com/benpate/rosetta/html"
)

// ContentMaxLength is the hard ceiling (in runes) on how much body content a
// single Stream or Activity may store.  It bounds `content.raw` and `content.html`
// in ContentSchema so that no write path -- local authoring, ActivityPub ingest, or
// import -- can persist an unbounded body and exhaust storage.  The `edit-content`
// step enforces its own, usually smaller, per-template limit up to this ceiling
// (see model/step.EditContent).
const ContentMaxLength = 1 << 20 // 1 MiB

// Content represents the WYSIWYG body content in a Stream or Activity
type Content struct {
	Format string `json:"format" bson:"format"`
	Raw    string `json:"raw"    bson:"raw"`
	HTML   string `json:"html"   bson:"html"`
}

// NewContent returns a fully initialized, empty Content
func NewContent() Content {
	return Content{}
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
