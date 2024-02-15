package model

import (
	"github.com/benpate/rosetta/html"
)

// Content represents the WYSIWYG body content in a Stream or Activity
type Content struct {
	Format string `json:"format" bson:"format"`
	Raw    string `json:"raw"    bson:"raw"`
	HTML   string `json:"html"   bson:"html"`
}

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
