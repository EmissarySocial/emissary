package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Template represents an HTML template to be used for generating an HTML page.
type Template struct {
	TemplateID  primitive.ObjectID    // Unique Identifier for this Template.
	URL         string                // URL where this template is published
	Category    string                // Human-readable category (grouping) used in management UI.
	Label       string                // Human-readable label used in management UI.
	IconURL     string                // Icon image used in management UI.
	Format      string                // TOKEN that other templates will use to reference this template.
	Schema      schema.Schema         // JSON Schema that describes the data required to populate this Template.
	Views       map[string]View       // Map of Views (by view.ID) that are available to Streams of this Template.
	States      map[string]State      // Map of States (by state.ID) that Streams of this Template can be in.
	Transitions map[string]Transition // Map of Transitions (by transition.ID) between States of this Template.

	journal.Journal
}

// ID returns the primary key of this object
func (t *Template) ID() string {
	return t.TemplateID.Hex()
}
