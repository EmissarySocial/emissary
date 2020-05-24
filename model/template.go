package model

import (
	"html/template"

	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
	"github.com/qri-io/jsonschema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Template represents an HTML template to be used for generating an HTML page.
type Template struct {
	TemplateID primitive.ObjectID // Unique Identifier for this Template.
	Category   string             // Human-readable category (grouping) used in management UI.
	Label      string             // Human-readable label used in management UI.
	IconURL    string             // Icon image used in management UI.
	Format     string             // TOKEN that other templates will use to reference this template.
	Form       string             // JSON-Form data that describes how to render an input form for this template.
	Schema     jsonschema.Schema  // JSON Schema that describes the data required to populate this template.
	Content    string             // String representation of the template.

	Compiled *template.Template // Compiled representation of the template.

	journal.Journal
}

// ID returns the primary key of this object
func (t *Template) ID() string {
	return t.TemplateID.Hex()
}

// Init parses all of the compiled elements of the Template after it has been loaded from a database.
func (t *Template) Init(funcMap map[string]interface{}) *derp.Error {

	compiled, err := template.New(t.Format).Funcs(funcMap).Parse(t.Content)

	if err != nil {
		return derp.New(500, "model.Template.Init", "Invalid Template Content", err)
	}

	// Save the compiled template into the model object
	t.Compiled = compiled

	return nil
}
