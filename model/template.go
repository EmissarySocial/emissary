package model

import (
	"html/template"

	"github.com/benpate/data/option"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
)

// Template represents an HTML template used for rendering Streams
type Template struct {
	TemplateID         string               `json:"templateId"         bson:"templateId"`         // Internal name/token other objects (like streams) will use to reference this Template.
	Role               string               `json:"role"               bson:"role"`               // Role that this Template performs in the system.  Used to match which streams can be contained by which other streams.
	Model              string               `json:"model"              bson:"model"`              // Type of model object that this template works with. (Stream, User, Group, Domain, etc.)
	Label              string               `json:"label"              bson:"label"`              // Human-readable label used in management UI.
	Description        string               `json:"description"        bson:"description"`        // Human-readable long-description text used in management UI.
	Category           string               `json:"category"           bson:"category"`           // Human-readable category (grouping) used in management UI.
	Icon               string               `json:"icon"               bson:"icon"`               // Icon image used in management UI.
	Sort               int                  `json:"sort"               bson:"sort"`               // Sort order used in management UI.
	ContainedBy        sliceof.String       `json:"containedBy"        bson:"containedBy"`        // Slice of Templates that can contain Streams that use this Template.
	ChildSortType      string               `json:"childSortType"      bson:"childSortType"`      // SortType used to display children
	ChildSortDirection string               `json:"childSortDirection" bson:"childSortDirection"` // Sort direction "asc" or "desc" (Default is ascending)
	URL                string               `json:"url"                bson:"url"`                // URL where this template is published
	Schema             schema.Schema        `json:"schema"             bson:"schema"`             // JSON Schema that describes the data required to populate this Template.
	States             mapof.Object[State]  `json:"states"             bson:"states"`             // Map of States (by state.ID) that Streams of this Template can be in.
	Roles              mapof.Object[Role]   `json:"roles"              bson:"roles"`              // Map of custom roles defined by this Template.
	Actions            mapof.Object[Action] `json:"actions"            bson:"actions"`            // Map of actions that can be performed on streams of this Template
	Bundles            mapof.Object[Bundle] `json:"bundles"            bson:"bundles"`            // Additional resources (JS, HS, CSS) reqired tp remder this Template.
	DefaultAction      string               `json:"defaultAction"      bson:"defaultAction"`      // Name of the action to be used when none is provided.  Also serves as the permissions for viewing a Stream.  If this is empty, it is assumed to be "view"
	HTMLTemplate       *template.Template
}

// NewTemplate creates a new, fully initialized Template object
func NewTemplate(templateID string, funcMap template.FuncMap) Template {

	return Template{
		TemplateID:         templateID,
		ContainedBy:        make([]string, 0),
		ChildSortType:      "rank",
		ChildSortDirection: option.SortDirectionAscending,
		States:             make(map[string]State),
		Roles:              make(map[string]Role),
		Actions:            make(map[string]Action),
		DefaultAction:      "view",
		HTMLTemplate:       template.New("").Funcs(funcMap),
	}
}

// ID implements the set.Value interface
func (template Template) ID() string {
	return template.TemplateID
}

// CanBeContainedBy returns TRUE if this Streams using this Template can be nested inside of
// Streams using the Template named in the parameters
func (template *Template) CanBeContainedBy(templateRoles ...string) bool {

	// Otherwise, this template MUSt list the potential parent Stream's *role* in its ContainedBy list
	for _, templateRole := range templateRoles {
		if slice.Contains(template.ContainedBy, templateRole) {
			return true
		}
	}
	return false
}

// State searches for the State in this Template that matches the provided StateID
// If found, it is returned along with a TRUE
// If not found, an empty state is returned along with a FALSE
func (template *Template) State(stateID string) (State, bool) {
	state, ok := template.States[stateID]
	return state, ok
}

// Action returns the action object for a specified name
func (template *Template) Action(actionID string) *Action {

	if action, ok := template.Actions[actionID]; ok {
		return &action
	}

	return nil
}

// Default returns the default Action for this Template.
func (template *Template) Default() *Action {
	return template.Action(template.DefaultAction)
}
