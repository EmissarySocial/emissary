package model

import (
	"html/template"
	"io/fs"

	"github.com/EmissarySocial/emissary/tools/templatemap"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/rosetta/translate"
)

// Template represents an HTML template used for building Streams
type Template struct {
	TemplateID         string               `json:"templateId"         bson:"templateId"`         // Internal name/token other objects (like streams) will use to reference this Template.
	URL                string               `json:"url"                bson:"url"`                // URL where this template is published
	TemplateRole       string               `json:"templateRole"       bson:"templateRole"`       // Role that this Template performs in the system.  Used to match which streams can be contained by which other streams.
	SocialRole         string               `json:"socialRole"         bson:"socialRole"`         // Role to use for this Template in social integrations (Article, Note, etc)
	SocialRules        translate.Pipeline   `json:"socialRules"        bson:"socialRules"`        // List of rules to convert this Template into a social object
	Model              string               `json:"model"              bson:"model"`              // Type of model object that this template works with. (Stream, User, Group, Domain, etc.)
	Extends            sliceof.String       `json:"extends"            bson:"extends"`            // List of templates that this template extends.  The first template in the list is the most important, and the last template in the list is the least important.
	ContainedBy        sliceof.String       `json:"containedBy"        bson:"containedBy"`        // Slice of Templates that can contain Streams that use this Template.
	Label              string               `json:"label"              bson:"label"`              // Human-readable label used in management UI.
	Description        string               `json:"description"        bson:"description"`        // Human-readable long-description text used in management UI.
	Category           string               `json:"category"           bson:"category"`           // Human-readable category (grouping) used in management UI.
	Icon               string               `json:"icon"               bson:"icon"`               // Icon image used in management UI.
	Sort               int                  `json:"sort"               bson:"sort"`               // Sort order used in management UI.
	ChildSortType      string               `json:"childSortType"      bson:"childSortType"`      // SortType used to display children
	ChildSortDirection string               `json:"childSortDirection" bson:"childSortDirection"` // Sort direction "asc" or "desc" (Default is ascending)
	WidgetLocations    sliceof.String       `json:"widget-locations"   bson:"widgetLocations"`    // List of locations where widgets can be placed.  Common values are: "TOP", "BOTTOM", "LEFT", "RIGHT"
	Schema             schema.Schema        `json:"schema"             bson:"schema"`             // JSON Schema that describes the data required to populate this Template.
	States             mapof.Object[State]  `json:"states"             bson:"states"`             // Map of States (by state.ID) that Streams of this Template can be in.
	AccessRoles        mapof.Object[Role]   `json:"accessRoles"        bson:"accessRoles"`        // Map of custom roles defined by this Template.
	Actions            mapof.Object[Action] `json:"actions"            bson:"actions"`            // Map of actions that can be performed on streams of this Template
	HTMLTemplate       *template.Template   `json:"-"                  bson:"-"`                  // Compiled HTML template
	TagPaths           []string             `json:"tagPaths"           bson:"tagPaths"`           // List of paths to tags that are used in this template
	SearchOptions      templatemap.Map      `json:"search"             bson:"search"`             // Compiled templates that override default search result values
	Bundles            mapof.Object[Bundle] `json:"bundles"            bson:"bundles"`            // Additional resources (JS, HS, CSS) reqired tp remder this Template.
	Resources          fs.FS                `json:"-"                  bson:"-"`                  // File system containing the template resources
	Datasets           DatasetMap           `json:"datasets"           bson:"-"`                  // Lookup codes defined by this template
	DefaultAction      string               `json:"defaultAction"      bson:"defaultAction"`      // Name of the action to be used when none is provided.  Also serves as the permissions for viewing a Stream.  If this is empty, it is assumed to be "view"
	Actor              StreamActor          `json:"actor"              bson:"actor"`              // ActivityPub Actor operated on behalf of this Template/Stream
}

type DatasetMap map[string]form.ReadOnlyLookupGroup

// NewTemplate creates a new, fully initialized Template object
func NewTemplate(templateID string, funcMap template.FuncMap) Template {

	return Template{
		TemplateID:         templateID,
		SocialRules:        make(translate.Pipeline, 0),
		Extends:            make([]string, 0),
		ContainedBy:        make([]string, 0),
		ChildSortType:      "rank",
		ChildSortDirection: option.SortDirectionAscending,
		WidgetLocations:    make(sliceof.String, 0),
		States:             make(map[string]State),
		AccessRoles:        make(map[string]Role),
		Actions:            make(map[string]Action),
		DefaultAction:      "view",
		HTMLTemplate:       template.New("").Funcs(funcMap),
		SearchOptions:      make(templatemap.Map),
		Bundles:            make(map[string]Bundle),
		Datasets:           make(DatasetMap),
	}
}

// ID implements the set.Value interface
func (template Template) ID() string {
	return template.TemplateID
}

func (template Template) IsZero() bool {
	if template.TemplateID != "" {
		return false
	} else if template.TemplateRole != "" {
		return false
	} else if len(template.Actions) > 0 {
		return false
	}

	return true
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

func (template *Template) IsValidWidgetLocation(location string) bool {
	return slice.Contains(template.WidgetLocations, location)
}

// State searches for the State in this Template that matches the provided StateID
// If found, it is returned along with a TRUE
// If not found, an empty state is returned along with a FALSE
func (template *Template) State(stateID string) (State, bool) {
	state, ok := template.States[stateID]
	return state, ok
}

// Action returns the action object for a specified name
func (template *Template) Action(actionID string) (Action, bool) {
	action, ok := template.Actions[actionID]
	return action, ok
}

// Default returns the default Action for this Template.
func (template *Template) Default() Action {
	return template.Actions[template.DefaultAction]
}

func (template *Template) Inherit(parent *Template) {

	// Null check.
	if parent == nil {
		return
	}

	// Inherit schema items from the parent.
	template.Schema.Inherit(parent.Schema)

	// Inherit WidgetLocations.
	if len(template.WidgetLocations) == 0 {
		template.WidgetLocations = parent.WidgetLocations
	}

	// Inherit TemplateRole.
	if template.TemplateRole == "" {
		template.TemplateRole = parent.TemplateRole
	}

	// Inherit SocialRole.
	if template.SocialRole == "" {
		template.SocialRole = parent.SocialRole
	}

	// Apply SocialRules
	if len(parent.SocialRules) > 0 {
		template.SocialRules = append(template.SocialRules, parent.SocialRules...)
	}

	// Inherit ContainedBy.
	if len(template.ContainedBy) == 0 {
		template.ContainedBy = parent.ContainedBy
	}

	// Inherit Model.
	if template.Model == "" {
		template.Model = parent.Model
	}

	// Inherit SearchTemplate
	for optionID, option := range parent.SearchOptions {
		if _, exists := template.SearchOptions[optionID]; !exists {
			template.SearchOptions[optionID] = option
		}
	}

	// Inherit Roles from the parent.
	for roleID, role := range parent.AccessRoles {
		if _, exists := template.AccessRoles[roleID]; !exists {
			template.AccessRoles[roleID] = role
		}
	}

	// Inherit States from the parent.
	for stateID, state := range parent.States {
		if _, exists := template.States[stateID]; !exists {
			template.States[stateID] = state
		}
	}

	// Inherit Actions from the parent.
	for actionID, action := range parent.Actions {
		if _, exists := template.Actions[actionID]; !exists {
			template.Actions[actionID] = action
		}
	}

	// Inherit HTMLTemplates from the parent.
	for _, templateName := range parent.HTMLTemplate.Templates() {
		if template.HTMLTemplate.Lookup(templateName.Name()) == nil {
			if _, err := template.HTMLTemplate.AddParseTree(templateName.Name(), templateName.Tree); err != nil {
				derp.Report(derp.Wrap(err, "model.Template.Inherit", "Error adding template", templateName.Name()))
			}
		}
	}
}

// IsSearch returns TRUE if this is Template is a search engine
func (template *Template) IsSearch() bool {
	return template.Model == "search"
}

/******************************************
 * OEmbed Methods
 ******************************************/

// HasOEmbed returns TRUE if this Template has an OEmbed template
func (template *Template) HasOEmbed() bool {
	return template.HTMLTemplate.Lookup("oembed") != nil
}

// GetOEmbed returns the OEmbed template for this Template
// If no OEmbed template is found, then nil is returned
func (template *Template) GetOEmbed() *template.Template {
	return template.HTMLTemplate.Lookup("oembed")
}
