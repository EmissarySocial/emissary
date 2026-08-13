package model

import (
	"html/template"
	"io/fs"
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
)

// Theme represents an HTML template used for building all hard-coded application elements (but not dynamic streams)
type Theme struct {
	ThemeID        string                  `json:"themeID"        bson:"themeID"`        // Internal name/token other objects (like streams) will use to reference this Theme.
	Extends        []string                `json:"extends"        bson:"extends"`        // List of other themes that this theme extends
	Category       string                  `json:"category"       bson:"category"`       // Category of this theme (for grouping)
	Label          string                  `json:"label"          bson:"label"`          // Human-readable label for this theme
	Description    string                  `json:"description"    bson:"description"`    // Human-readable description for this theme
	Rank           int                     `json:"rank"           bson:"rank"`           // Sort order for this theme
	HTMLTemplate   *template.Template      `json:"-"              bson:"-"`              // HTML template for this theme
	Bundles        mapof.Object[Bundle]    `json:"bundles"        bson:"bundles"`        // // Additional resources (JS, HS, CSS) reqired tp remder this Theme.
	Resources      fs.FS                   `json:"-"              bson:"-"`              // File system containing the template resources
	Datasets       mapof.Object[mapof.Any] `json:"datasets"       bson:"datasets"`       // Datasets used by this theme
	StartupTasks   []form.LookupCode       `json:"startupTasks"   bson:"startupTasks"`   // A list of tasks to be completed at startup
	StartupStreams []mapof.Any             `json:"startupStreams" bson:"startupStreams"` // Dataset of Streams to initialize when this theme is first chosen.
	StartupGroups  []mapof.Any             `json:"startupGroups"  bson:"startupGroups"`  // Dataset of Groups to initialize when this theme is first chosen.
	DefaultFolders []mapof.Any             `json:"defaultFolders" bson:"defaultFolders"` // Dataset of Folders to initialize when a User is added using this Theme.
	DefaultInbox   string                  `json:"defaultInbox"   bson:"defaultInbox"`   // Default Inbox Template for Users created underneath this theme
	DefaultOutbox  string                  `json:"defaultOutbox"  bson:"defaultOutbox"`  // Default Outbox Template for Users created underneath this theme
	Form           form.Element            `json:"form"           bson:"form"`           // Form used to edit custom data
	Schema         schema.Schema           `json:"schema"         bson:"schema"`         // Schema used to validate custom data
	Data           mapof.String            `json:"data"           bson:"data"`           // Custom data for this theme
	IsVisible      bool                    `json:"isVisible"      bson:"isVisible"`      // Is this theme visible to the site owners?
}

// NewTheme creates a new, fully initialized Theme object
func NewTheme(templateID string, funcMap template.FuncMap) Theme {

	return Theme{
		ThemeID:        templateID,
		Extends:        make([]string, 0),
		HTMLTemplate:   template.New("").Funcs(funcMap),
		Bundles:        mapof.NewObject[Bundle](),
		Datasets:       mapof.NewObject[mapof.Any](),
		StartupTasks:   make([]form.LookupCode, 0),
		StartupStreams: make([]mapof.Any, 0),
		StartupGroups:  make([]mapof.Any, 0),
		DefaultFolders: make([]mapof.Any, 0),
		DefaultInbox:   "user-inbox",
		DefaultOutbox:  "user-outbox",
		Form:           form.NewElement(),
		Schema:         schema.Schema{},
		Data:           mapof.NewString(),
	}
}

func (theme Theme) LookupCode() form.LookupCode {
	return form.LookupCode{
		Value:       theme.ThemeID,
		Label:       theme.Label,
		Description: theme.Description,
	}
}

// StartupStreamTokens returns the "token" of every Stream that this Theme defines in
// "startupStreams".  It names the complete set that a caller may ask Stream.Startup to create,
// which is what a caller with no user-supplied selection of its own passes through.
func (theme Theme) StartupStreamTokens() sliceof.String {

	result := make(sliceof.String, 0, len(theme.StartupStreams))

	for _, startupStream := range theme.StartupStreams {
		result = append(result, startupStream.GetString("token"))
	}

	return result
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

// SortThemes is a sort.Slice function that sorts themes by their label
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
			if _, err := theme.HTMLTemplate.AddParseTree(templateName.Name(), templateName.Tree); err != nil {
				derp.Report(derp.Wrap(err, "model.Theme.Inherit", "Adding template", templateName.Name()))
			}
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
