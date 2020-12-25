package model

import (
	"bytes"
	"html/template"

	"github.com/benpate/derp"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
)

// View is an individual HTML template that can render a part of a stream
type View struct {
	ViewID string   `json:"viewId" bson:"viewId"` // Unique Identifier for this view.
	Label  string   `json:"label"  bson:"label"`  // Human-friendly label for this view.
	Roles  []string `json:"roles"  bson:"roles"`  // Roles that can access this view.  If empty, then no additional restrictions.
	HTML   string   `json:"html"   bson:"html"`   // Raw html.Template contents to render

	compiled *template.Template // Parsed HTML template to render (by merging with Stream dataset)
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

// Compiled calculates or retrieves the compiled state of this view.
func (v *View) Compiled() (*template.Template, error) {

	// If this view has already been compiled, then return the compiled version
	if v.compiled == nil {

		// Try to minify the incoming template... (this should be moved to a different place.)
		m := minify.New()
		m.AddFunc("text/html", html.Minify)

		minified, err := m.String("text/html", v.HTML)

		if err != nil {
			return nil, derp.Wrap(err, "ghost.model.View.Compiled", "Error minifying template", v)
		}

		result, err := template.New(v.ViewID).Parse(minified)

		if err != nil {
			return nil, derp.Wrap(err, "ghost.model.View.Compiled", "Unable to parse template HTML", v)
		}

		// Save the value into this view
		v.compiled = result
	}

	return v.compiled, nil
}

// Execute executes this template on the provided data.  It maintains a cache of the compiled template
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
