package model

import (
	"html/template"
)

// View is an individual HTML template that can render a part of a stream
type View struct {
	ViewID   string             `json:"viewId" bson:"viewId"` // Unique Identifier for this view.
	Roles    []string           `json:"roles"  bson:"roles"`  // Roles that can access this view.  If empty, then no additional restrictions.
	Template *template.Template `json:"-"      bson:"-"`      // In-Memory data structure for the compiled HTML template
}

// NewView returns a fully populated View object.
func NewView() View {
	return View{
		Roles: make([]string, 0),
	}
}

// MatchRoles returns TRUE if one or more of the provided roles matches the requirements for this View.
// If no roles are defined for this View, then access is always granted.
func (v View) MatchRoles(roles ...string) bool {

	for i := range v.Roles {
		for j := range roles {
			if roles[i] == v.Roles[j] {
				return true
			}
		}
	}

	return false
}
