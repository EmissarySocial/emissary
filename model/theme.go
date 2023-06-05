package model

import (
	"html/template"
	"io/fs"

	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
)

// Theme represents an HTML template used for rendering all hard-coded application elements (but not dynamic streams)
type Theme struct {
	ThemeID      string               `json:"themeID"     bson:"themeID"`     // Internal name/token other objects (like streams) will use to reference this Theme.
	Label        string               `json:"label"       bson:"label"`       // Human-readable label for this theme
	Description  string               `json:"description" bson:"description"` // Human-readable description for this theme
	Rank         int                  `json:"rank"        bson:"rank"`        // Sort order for this theme
	HTMLTemplate *template.Template   `json:"-"           bson:"-"`           // HTML template for this theme
	Bundles      mapof.Object[Bundle] `json:"bundles"     bson:"bundles"`     // // Additional resources (JS, HS, CSS) reqired tp remder this Theme.
	Resources    fs.FS                `json:"-"           bson:"-"`           // File system containing the template resources
}

// NewTheme creates a new, fully initialized Theme object
func NewTheme(templateID string, funcMap template.FuncMap) Theme {

	return Theme{
		ThemeID:      templateID,
		Bundles:      mapof.NewObject[Bundle](),
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
