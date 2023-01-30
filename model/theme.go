package model

import (
	"html/template"

	"github.com/benpate/rosetta/mapof"
)

// Theme represents an HTML template used for rendering all hard-coded application elements (but not dynamic streams)
type Theme struct {
	ThemeID      string       `json:"themeID" bson:"themeID"` // Internal name/token other objects (like streams) will use to reference this Theme.
	Bundles      mapof.String `json:"bundles" bson:"bundles"` // Map of bundles that are required to render this theme
	HTMLTemplate *template.Template
}

// NewTheme creates a new, fully initialized Theme object
func NewTheme(templateID string, funcMap template.FuncMap) Theme {

	return Theme{
		ThemeID:      templateID,
		Bundles:      mapof.NewString(),
		HTMLTemplate: template.New("").Funcs(funcMap),
	}
}
