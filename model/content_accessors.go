package model

import "github.com/benpate/rosetta/schema"

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

/********************************
 * Getter/Setter Interfaces
 ********************************/

func (content *Content) GetStringOK(name string) (string, bool) {
	switch name {

	case "format":
		return content.Format, true

	case "raw":
		return content.Raw, true

	case "html":
		return content.HTML, true

	default:
		return "", false
	}
}

func (content *Content) SetString(name string, value string) bool {
	switch name {

	case "format":
		content.Format = value
		return true

	case "raw":
		content.Raw = value
		return true

	case "html":
		content.HTML = value
		return true

	default:
		return false
	}
}
