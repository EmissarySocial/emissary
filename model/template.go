package model

import (
	"github.com/benpate/derp"
	"github.com/benpate/schema"
)

// Template represents an HTML template to be used for generating an HTML page.
type Template struct {
	TemplateID  string           `json:"templateId"  bson:"templateId"`  // Internal name/token other objects (like streams) will use to reference this Template.
	Label       string           `json:"label"       bson:"label"`       // Human-readable label used in management UI.
	Description string           `json:"description" bson:"description"` // Human-readable long-description text used in management UI.
	Category    string           `json:"category"    bson:"category"`    // Human-readable category (grouping) used in management UI.
	IconURL     string           `json:"iconUrl"     bson:"iconUrl"`     // Icon image used in management UI.
	URL         string           `json:"url"         bson:"url"`         // URL where this template is published
	Schema      *schema.Schema   `json:"schema"      bson:"schema"`      // JSON Schema that describes the data required to populate this Template.
	States      map[string]State `json:"states"      bson:"states"`      // Map of States (by state.ID) that Streams of this Template can be in.
	Views       map[string]View  `json:"views"       bson:"views"`       // Map of Views (by view.ID) that are available to Streams of this Template.
}

func NewTemplate(templateID string) *Template {
	return &Template{
		TemplateID: templateID,
		States:     make(map[string]State),
		Views:      make(map[string]View),
	}
}

// View locates and verifies a state/view combination.
func (template Template) View(stateName string, viewName string) (*View, *derp.Error) {

	// If no view name is specified, then use "DEFAULT" instead.
	if viewName == "" {
		viewName = "default"
	}

	// Verify that the requested State exists
	if state, ok := template.States[stateName]; ok {

		// Scan all views in the state to verify that this view is allowed in this State
		for _, allowedViewName := range state.Views {

			// If the view is allowed for this State
			if viewName == allowedViewName {

				// Try to find this view in the template
				if view, ok := template.Views[viewName]; ok {
					return &view, nil
				}
			}
		}
	}

	// If we're not trying the default view, then switch to that view now.
	if viewName != "default" {
		return template.View(stateName, "default")
	}

	return nil, derp.New(404, "ghost.model.Template.View", "Unrecognized State", template, stateName)
}

func (template Template) Transition(stateID string, transitionID string) *Transition {

	if state, ok := template.States[stateID]; ok {

		for index := range state.Transitions {
			if state.Transitions[index].ID == transitionID {
				return &(template.States[stateID].Transitions[index])
			}
		}
	}

	return nil
}

// Populate safely copies values from an external Template into this one.
func (template *Template) Populate(from *Template) {

	template.Label = template.BestString(template.Label, from.Label)
	template.Description = template.BestString(template.Description, from.Description)
	template.Category = template.BestString(template.Category, from.Category)
	template.IconURL = template.BestString(template.IconURL, from.IconURL)
	template.URL = template.BestString(template.URL, from.URL)

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
}

func (template Template) BestString(local string, remote string) string {

	if local != "" {
		return local
	}

	return remote
}
