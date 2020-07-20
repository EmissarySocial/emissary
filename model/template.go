package model

import (
	"github.com/benpate/schema"
)

// Template represents an HTML template to be used for generating an HTML page.
type Template struct {
	TemplateID  string                `json:"name"        bson:"name"`        // Internal name/token other objects (like streams) will use to reference this Template.
	Label       string                `json:"label"       bson:"label"`       // Human-readable label used in management UI.
	Description string                `json:"description" bson:"description"` // Human-readable long-description text used in management UI.
	Category    string                `json:"category"    bson:"category"`    // Human-readable category (grouping) used in management UI.
	IconURL     string                `json:"iconUrl"     bson:"iconUrl"`     // Icon image used in management UI.
	URL         string                `json:"url"         bson:"url"`         // URL where this template is published
	Schema      schema.Element        `json:"schema"      bson:"schema"`      // JSON Schema that describes the data required to populate this Template.
	States      map[string]State      `json:"states"      bson:"states"`      // Map of States (by state.ID) that Streams of this Template can be in.
	Transitions map[string]Transition `json:"transitions" bson:"transitions"` // Map of Transitions (by transition.ID) between States of this Template.
	Views       map[string]View       `json:"views"       bson:"views"`       // Map of Views (by view.ID) that are available to Streams of this Template.
}

// View returns a specific View that is defined in this Template.  If the requested view does not exist, then
// the "default" View is returned.  If there is no default, then an empty View is returned along with a FALSE.
func (t *Template) View(name string) (View, bool) {

	if name != "" {
		if view, ok := t.Views[name]; ok {
			return view, true
		}
	}

	if view, ok := t.Views["default"]; ok {
		return view, true
	}

	return View{}, false
}
