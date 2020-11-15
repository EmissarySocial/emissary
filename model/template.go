package model

import (
	"github.com/benpate/choose"
	"github.com/benpate/data/compare"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/path"
	"github.com/benpate/schema"
)

// Template represents an HTML template to be used for generating an HTML page.
type Template struct {
	TemplateID  string               `json:"templateId"  bson:"templateId"`  // Internal name/token other objects (like streams) will use to reference this Template.
	Label       string               `json:"label"       bson:"label"`       // Human-readable label used in management UI.
	Description string               `json:"description" bson:"description"` // Human-readable long-description text used in management UI.
	Category    string               `json:"category"    bson:"category"`    // Human-readable category (grouping) used in management UI.
	IconURL     string               `json:"iconUrl"     bson:"iconUrl"`     // Icon image used in management UI.
	ContainedBy []string             `json:"containedBy" bson:"containedBy"` // Slice of Templates that can contain Streams that use this Template.
	URL         string               `json:"url"         bson:"url"`    // URL where this template is published
	Schema      *schema.Schema       `json:"schema"      bson:"schema"` // JSON Schema that describes the data required to populate this Template.
	States      map[string]State     `json:"states"      bson:"states"` // Map of States (by state.ID) that Streams of this Template can be in.
	Views       []View               `json:"views"       bson:"views"`  // Map of Views (by view.ID) that are available to Streams of this Template.
	Forms       map[string]form.Form `json:"forms"       bson:"forms"`  // Map of Forms (by form.ID) that are available in transitions between states.
}

// NewTemplate creates a new, fully initialized Template object
func NewTemplate(templateID string) *Template {
	return &Template{
		TemplateID: templateID,
		ContainedBy: make([]string, 0),
		States:     make(map[string]State),
		Views:      make([]View, 0),
		Forms:      make(map[string]form.Form),
	}
}


// CanBeContainedBy returns TRUE if this Streams using this Template can be nested inside of
// Streams using the Template named in the parameters
func (template Template) CanBeContainedBy(templateName string) bool {

	return compare.Contains(template.ContainedBy, templateName)
}

// View locates and verifies a state/view combination.
func (template Template) View(stateName string, viewName string) (*View, error) {

	var showView string

	if viewName == "" {
		viewName = "default"
	}

	// Verify that the requested State exists
	if state, ok := template.States[stateName]; ok {

		for _, view := range state.Views {

			// TODO: Check permissions

			if view == viewName {
				showView = view
				break
			}

			if showView == "" {
				showView = view
			}
		}

		if showView != "" {
			for _, view := range template.Views {
				if view.Name == showView {
					return &view, nil
				}
			}
		}

		return nil, derp.New(500, "ghost.model.Template.View", "Unauthorized", stateName, viewName)
	}

	return nil, derp.New(404, "ghost.model.Template.View", "Unrecognized State", template, stateName)
}

// Transition returns the Transition for a particular State/Transition combination.
func (template Template) Transition(stateID string, transitionID string) (*Transition, error) {

	if state, ok := template.States[stateID]; ok {

		if transition, ok := state.Transitions[transitionID]; ok {
			return &transition, nil
		}
	}

	return nil, derp.New(404, "ghost.model.template.Transition", "Unrecognized StateID", stateID, transitionID)
}

// Form returns the Form for a particular State/Transition combination.
func (template Template) Form(stateID string, transitionID string) (*form.Form, error) {

	transition, err := template.Transition(stateID, transitionID)

	if err != nil {
		return nil, derp.Wrap(err, "ghost.model.template.Form", "Unable to locate transition", stateID, transitionID)
	}

	if form, ok := template.Forms[transition.Form]; ok {
		return &form, nil
	}

	return nil, derp.New(404, "ghost.model.template.Form", "Undefined form", transition.Form)
}

// Populate safely copies values from an external Template into this one.
func (template *Template) Populate(from *Template) {

	template.Label = choose.String(template.Label, from.Label)
	template.Description = choose.String(template.Description, from.Description)
	template.Category = choose.String(template.Category, from.Category)
	template.IconURL = choose.String(template.IconURL, from.IconURL)
	template.URL = choose.String(template.URL, from.URL)

	if len(from.ContainedBy) > 0 {
		template.ContainedBy = append(template.ContainedBy, from.ContainedBy...)
	}

	if template.Schema == nil {
		template.Schema = from.Schema
	}

	if from.States != nil {
		for name, state := range from.States {
			template.States[name] = state
		}
	}

	if from.Views != nil {
		for _, view := range from.Views {
			template.Views = append(template.Views, view)
		}
	}

	if from.Forms != nil {
		for name, form := range from.Forms {
			template.Forms[name] = form
		}
	}
}

func (template Template) GetPath(p path.Path) (interface{}, error) {

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
