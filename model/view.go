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
	Label       string             `json:"label"`       // Human-friendly label for this view.
	Name        string             `json:"name"`        // Name of the file in the template package where HTML is stored (without extension).
	Permissions []string           `json:"permissions"` // View permissions (to implement later)
	HTML        string             `json:"html"`        // Raw HTML to render
	compiled    *template.Template `json:"-"`           // Parsed HTML template to render (by merging with Stream dataset)
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
			return nil, derp.Wrap(err, "model.View.Template", "Error minifying template", v.Name)
		}

		result, err := template.New("").Parse(minified)

		if err != nil {
			return nil, derp.Wrap(err, "model.View.Template", "Unable to parse template HTML", v.Name)
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
		return "", derp.Wrap(err, "Model.View.Template", "Error executing template", v.HTML, data)
	}

	// Return to caller.  TRUE means that the object has been changed.
	return buffer.String(), nil
}
