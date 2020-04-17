package model

import (
	"html/template"

	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Template represents an HTML template to be used for generating an HTML page.
type Template struct {
	TemplateID primitive.ObjectID
	Label      string
	Category   string
	IconURL    string
	Content    template.Template

	journal.Journal
}

// ID returns the primary key of this object
func (template *Template) ID() string {
	return template.TemplateID.Hex()
}
