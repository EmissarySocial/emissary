package model

import "github.com/benpate/rosetta/schema"

// ContentSchema returns the JSON Schema for a Content object
func ContentSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"format": schema.String{Enum: []string{ContentFormatHTML, ContentFormatEditorJS, ContentFormatMarkdown, ContentFormatText}},
			"raw":    schema.String{Format: "unsafe-any"},
			"html":   schema.String{Format: "html"},
		},
	}
}

/********************************
 * Getter/Setter Interfaces
 ********************************/

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
