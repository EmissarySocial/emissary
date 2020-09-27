package model

import (
	"github.com/benpate/choose"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/schema"
)

// Template represents an HTML template to be used for generating an HTML page.
type Template struct {
	TemplateID  string               `json:"templateId"  bson:"templateId"`  // Internal name/token other objects (like streams) will use to reference this Template.
	Label       string               `json:"label"       bson:"label"`       // Human-readable label used in management UI.
	Description string               `json:"description" bson:"description"` // Human-readable long-description text used in management UI.
	Category    string               `json:"category"    bson:"category"`    // Human-readable category (grouping) used in management UI.
	IconURL     string               `json:"iconUrl"     bson:"iconUrl"`     // Icon image used in management UI.
	URL         string               `json:"url"         bson:"url"`         // URL where this template is published
	Schema      *schema.Schema       `json:"schema"      bson:"schema"`      // JSON Schema that describes the data required to populate this Template.
	States      map[string]State     `json:"states"      bson:"states"`      // Map of States (by state.ID) that Streams of this Template can be in.
	Views       map[string]View      `json:"views"       bson:"views"`       // Map of Views (by view.ID) that are available to Streams of this Template.
	Forms       map[string]form.Form `json:"forms"       bson:"forms"`       // Map of Forms (by form.ID) that are available in transitions between states.
}

// NewTemplate creates a new, fully initialized Template object
func NewTemplate(templateID string) *Template {
	return &Template{
		TemplateID: templateID,
		States:     make(map[string]State),
		Views:      make(map[string]View),
		Forms:      make(map[string]form.Form),
	}
}

// View locates and verifies a state/view combination.
func (template Template) View(stateName string, viewName string) (*View, error) {

	// If no view name is specified, then use "DEFAULT" instead.
	if viewName == "" {
		viewName = "default"
	}

	// Verify that the requested State exists
	if state, ok := template.States[stateName]; ok {

		// Verify that this is an allowed view
		if _, ok := state.Views[viewName]; ok {

			// TODO: Check permissions here

			if view, ok := template.Views[viewName]; ok {
				return &view, nil
			}
		}

		// If we're not trying the default view, then switch to that view now.
		if viewName != "default" {
			return template.View(stateName, "default")
		}

		return nil, derp.New(500, "ghost.model.TemplateView", "Unauthorized", stateName, viewName)
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

	if template.Schema == nil {
		template.Schema = from.Schema
	}

	if from.States != nil {
		for name, state := range from.States {
			template.States[name] = state
		}
	}

	if from.Views != nil {
		for name, view := range from.Views {
			template.Views[name] = view
		}
	}

	if from.Forms != nil {
		for name, form := range from.Forms {
			template.Forms[name] = form
		}
	}
}
