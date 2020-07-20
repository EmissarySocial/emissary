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
	Schema      schema.Schema    `json:"schema"      bson:"schema"`      // JSON Schema that describes the data required to populate this Template.
	States      map[string]State `json:"states"      bson:"states"`      // Map of States (by state.ID) that Streams of this Template can be in.
	Views       map[string]View  `json:"views"       bson:"views"`       // Map of Views (by view.ID) that are available to Streams of this Template.
}

// View locates and verifies a state/view combination.
func (template Template) View(stateName string, viewName string) (View, *derp.Error) {

	// Verify that the requested State exists
	if state, ok := template.States[stateName]; ok {

		// Scan all views in the state to verify that this view is allowed in this State
		for _, allowedViewName := range state.Views {

			// If the view is allowed for this State
			if viewName == allowedViewName {

				// Try to find this view in the template
				if view, ok := template.Views[viewName]; ok {
					return view, nil
				}
			}
		}
	}

	return View{}, derp.New(404, "ghost.model.Template.View", "Unrecognized State", stateName)
}
