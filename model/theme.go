package model

import (
	"html/template"

	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
)

// Theme represents an HTML template used for rendering all hard-coded application elements (but not dynamic streams)
type Theme struct {
	ThemeID      string       `json:"themeID"     bson:"themeID"`     // Internal name/token other objects (like streams) will use to reference this Theme.
	Label        string       `json:"label"       bson:"label"`       // Human-readable label for this theme
	Description  string       `json:"description" bson:"description"` // Human-readable description for this theme
	Rank         int          `json:"rank"        bson:"rank"`        // Sort order for this theme
	Bundles      mapof.String `json:"bundles"     bson:"bundles"`     // Map of bundles that are required to render this theme
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

func (theme Theme) LookupCode() form.LookupCode {
	return form.LookupCode{
		Value:       theme.ThemeID,
		Label:       theme.Label,
		Description: theme.Description,
	}
}
