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

// MatchAnonymous returns TRUE if this View does not require any special permissions.
func (v View) MatchAnonymous() bool {
	return len(v.Roles) == 0
}

// MatchRoles returns TRUE if one or more of the provided roles matches the requirements for this View.
// If no roles are defined for this View, then access is always granted.
func (v View) MatchRoles(roles ...string) bool {

	if v.MatchAnonymous() {
		return true
	}

	for i := range roles {
		for j := range v.Roles {
			if roles[i] == v.Roles[j] {
				return true
			}
		}
	}

	return false
}

/* Execute executes this template on the provided data.  It maintains a cache of the compiled template
func (v *View) Execute(data interface{}) (string, error) {

	var buffer bytes.Buffer

	template, err := v.Compiled()

	if err != nil {
		return "", derp.Wrap(err, "ghost.model.View.Execute", "Error gettin compiled template")
	}

	if err := template.Execute(&buffer, data); err != nil {
		return "", derp.Wrap(err, "ghost.model.View.Execute", "Error executing template", v.HTML, data)
	}

	// Return to caller.  TRUE means that the object has been changed.
	return buffer.String(), nil
}
*/
