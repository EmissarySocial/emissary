package model

import (
	"html/template"

	"github.com/benpate/compare"
	"github.com/benpate/derp"
	"github.com/benpate/path"
	"github.com/benpate/schema"
)

// Template represents an HTML template to be used for generating an HTML page.
type Template struct {
	TemplateID  string                        `json:"templateId"    bson:"templateId"`  // Internal name/token other objects (like streams) will use to reference this Template.
	Label       string                        `json:"label"         bson:"label"`       // Human-readable label used in management UI.
	Description string                        `json:"description"   bson:"description"` // Human-readable long-description text used in management UI.
	Category    string                        `json:"category"      bson:"category"`    // Human-readable category (grouping) used in management UI.
	IconURL     string                        `json:"iconUrl"       bson:"iconUrl"`     // Icon image used in management UI.
	ContainedBy []string                      `json:"containedBy"   bson:"containedBy"` // Slice of Templates that can contain Streams that use this Template.
	URL         string                        `json:"url"           bson:"url"`         // URL where this template is published
	Schema      *schema.Schema                `json:"schema"        bson:"schema"`      // JSON Schema that describes the data required to populate this Template.
	States      map[string]State              `json:"states"        bson:"states"`      // Map of States (by state.ID) that Streams of this Template can be in.
	Roles       map[string]Role               `json:"roles"         bson:"roles"`       // Map of custom roles defined by this Template.
	Actions     map[string]Action             `json:"actions"       bson:"actions"`     // Map of actions that can be performed on streams of this Template
	Files       map[string]*template.Template `json:"files"         bson:"files"`       // Map of the HTML files that comprise this Template
}

// NewTemplate creates a new, fully initialized Template object
func NewTemplate(templateID string) Template {
	return Template{
		TemplateID:  templateID,
		ContainedBy: make([]string, 0),
		States:      make(map[string]State),
		Roles:       make(map[string]Role),
		Actions:     make(map[string]Action),
		Files:       make(map[string]*template.Template),
	}
}

// CanBeContainedBy returns TRUE if this Streams using this Template can be nested inside of
// Streams using the Template named in the parameters
func (template *Template) CanBeContainedBy(templateName string) bool {
	return compare.Contains(template.ContainedBy, templateName)
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

// HTMLTemplate returns a named html/template
func (template *Template) HTMLTemplate(filename string) (*template.Template, bool) {
	result, ok := template.Files[filename]
	return result, ok
}

// GetPath implements the path.Getter interface.
func (template *Template) GetPath(p path.Path) (interface{}, error) {

	switch p.Head() {

	case "templateId":
		return template.TemplateID, nil
	case "category":
		return template.Label, nil
	case "containedBy":
		return template.ContainedBy, nil
	case "label":
		return template.Label, nil
	}

	return nil, derp.New(500, "ghost.model.Template.GetPath", "Unrecognized Path", p)
}

// Validate runs any post-processing required after a Template is parsed by the TemplateService
func (template *Template) Validate() {

	for actionID, action := range template.Actions {
		action.ActionID = actionID
		action.Validate()
		template.Actions[actionID] = action
	}
}
