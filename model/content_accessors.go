package model

import (
	"github.com/EmissarySocial/emissary/tools/markdown"
	"github.com/benpate/rosetta/schema"
)

// ContentSchema returns the JSON Schema for a Content object
func ContentSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"format": schema.String{Enum: []string{ContentFormatHTML, ContentFormatEditorJS, ContentFormatMarkdown, ContentFormatText}},
			"raw":    schema.String{Format: "unsafe-any", MaxLength: ContentMaxLength},
			// The application's own sanitizing policy, which keeps the classes and iframes that
			// rosetta's "html" format strips.  Every Stream template inherits this (BUG-278)
			"html": schema.String{Format: markdown.SchemaFormat, MaxLength: ContentMaxLength},
		},
	}
}

/********************************
 * Getter/Setter Interfaces
 ********************************/

// GetPointer returns a pointer to the named property. Implements schema.PointerGetter.
func (content *Content) GetPointer(name string) (any, bool) {

	switch name {

	case "format":
		return &content.Format, true

	case "raw":
		return &content.Raw, true

	case "html":
		return &content.HTML, true

	}

	return nil, false
}
