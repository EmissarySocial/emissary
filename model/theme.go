package model

import (
	"html/template"
	"io/fs"
	"strings"

	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
)

// Theme represents an HTML template used for rendering all hard-coded application elements (but not dynamic streams)
type Theme struct {
	ThemeID        string                  `json:"themeID"        bson:"themeID"`        // Internal name/token other objects (like streams) will use to reference this Theme.
	Extends        []string                `json:"extends" bson:"extends"`               // List of other themes that this theme extends
	Category       string                  `json:"category"       bson:"category"`       // Category of this theme (for grouping)
	Label          string                  `json:"label"          bson:"label"`          // Human-readable label for this theme
	Description    string                  `json:"description"    bson:"description"`    // Human-readable description for this theme
	Rank           int                     `json:"rank"           bson:"rank"`           // Sort order for this theme
	HTMLTemplate   *template.Template      `json:"-"              bson:"-"`              // HTML template for this theme
	Bundles        mapof.Object[Bundle]    `json:"bundles"        bson:"bundles"`        // // Additional resources (JS, HS, CSS) reqired tp remder this Theme.
	Resources      fs.FS                   `json:"-"              bson:"-"`              // File system containing the template resources
	Datasets       mapof.Object[mapof.Any] `json:"datasets"       bson:"datasets"`       // Datasets used by this theme
	StartupStreams []mapof.Any             `json:"startupStreams" bson:"startupStreams"` // Dataset of Streams to initialize when this theme is first chosen.
	StartupGroups  []mapof.Any             `json:"startupGroups"  bson:"startupGroups"`  // Dataset of Groups to initialize when this theme is first chosen.
	DefaultFolders []mapof.Any             `json:"defaultFolders" bson:"defaultFolders"` // Dataset of Folders to initialize when a User is added using this Theme.
	IsVisible      bool                    `json:"isVisible"      bson:"isVisible"`      // Is this theme visible to the site owners?
}

// NewTheme creates a new, fully initialized Theme object
func NewTheme(templateID string, funcMap template.FuncMap) Theme {

	return Theme{
		ThemeID:        templateID,
		Bundles:        mapof.NewObject[Bundle](),
		Datasets:       mapof.NewObject[mapof.Any](),
		StartupStreams: make([]mapof.Any, 0),
		StartupGroups:  make([]mapof.Any, 0),
		DefaultFolders: make([]mapof.Any, 0),
		HTMLTemplate:   template.New("").Funcs(funcMap),
	}
}

func (theme Theme) LookupCode() form.LookupCode {
	return form.LookupCode{
		Value:       theme.ThemeID,
		Label:       theme.Label,
		Description: theme.Description,
	}
}

func (theme Theme) IsEmpty() bool {
	if theme.ThemeID == "" {
		return true
	}

	if theme.HTMLTemplate == nil {
		return true
	}

	return false
}

// IsPlaceholder is a temporary function the SHOULD
// be removed once we have a sufficient number of
// well-defined themes.  Until then, it's used to
// mark themes that are in the system but don't work yet.
func (theme Theme) IsPlaceholder() bool {
	return strings.HasSuffix(theme.Label, "(TBD)")
}

func SortThemes(a, b Theme) bool {
	return a.Label < b.Label
}

func (theme *Theme) Inherit(parent *Theme) {

	// Null check.
	if parent == nil {
		return
	}

	// Inherit category from the parent (if not already defined)
	if theme.Category == "" {
		theme.Category = parent.Category
	}

	// Inherit Rank from the parent (if not already defined)
	if theme.Rank == 0 {
		theme.Rank = parent.Rank
	}

	// Inherit HTMLTemplates from the parent (if not already defined)
	for _, templateName := range parent.HTMLTemplate.Templates() {
		if theme.HTMLTemplate.Lookup(templateName.Name()) == nil {
			theme.HTMLTemplate.AddParseTree(templateName.Name(), templateName.Tree)
		}
	}

	// Inherit datasets from the parent (if not already defined)
	for key, value := range parent.Datasets {
		if _, ok := theme.Datasets[key]; !ok {
			theme.Datasets[key] = value
		}
	}

	// Inherit startup streams from the parent (if not already defined)
	if len(theme.StartupStreams) == 0 {
		theme.StartupStreams = parent.StartupStreams
	}

	// Inherit startup groups from the parent (if not already defined)
	if len(theme.StartupGroups) == 0 {
		theme.StartupGroups = parent.StartupGroups
	}

	// Inherit default folders from the parent (if not already defined)
	if len(theme.DefaultFolders) == 0 {
		theme.DefaultFolders = parent.DefaultFolders
	}
}
