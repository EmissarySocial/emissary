package model

import "html/template"

// View is an individual HTML template that can render a part of a stream
type View struct {
	Label       string            `json:"label"`       // Human-friendly label of this view
	Permissions []string          `json:"permissions"` // List of roles/users who can view this view
	HTML        string            `json:"html"`        // Raw HTML to render
	Template    template.Template `json:"-"`           // Parsed HTML template to render (by merging with Stream dataset)
}
