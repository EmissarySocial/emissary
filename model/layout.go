package model

import (
	"html/template"

	"github.com/benpate/rosetta/schema"
)

// Layout represents an HTML template used for rendering all hard-coded application elements (but not dynamic streams)
type Layout struct {
	LayoutID     string            `json:"layoutID"           bson:"layoutID"` // Internal name/token other objects (like streams) will use to reference this Layout.
	Schema       schema.Schema     `json:"schema"             bson:"schema"`   // JSON Schema that describes the data required to populate this Layout.
	Actions      map[string]Action `json:"actions"            bson:"actions"`  // Map of actions that can be performed on streams of this Layout
	HTMLTemplate *template.Template
}

// NewLayout creates a new, fully initialized Layout object
func NewLayout(templateID string, funcMap template.FuncMap) Layout {

	return Layout{
		LayoutID:     templateID,
		Actions:      make(map[string]Action),
		Schema:       schema.Schema{},
		HTMLTemplate: template.New("").Funcs(funcMap),
	}
}

// Action returns the action object for a specified name
func (layout *Layout) Action(actionID string) *Action {

	if action, ok := layout.Actions[actionID]; ok {
		return &action
	}

	result := NewAction()

	return &result
}
