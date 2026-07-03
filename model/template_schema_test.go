package model

import (
	"strings"
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

// TestTemplate_UnsupportedSchemaProperties_Clean confirms that a Stream template whose
// schema only declares properties the Stream model actually backs reports NO orphans.
func TestTemplate_UnsupportedSchemaProperties_Clean(t *testing.T) {

	template := NewTemplate("test-clean", nil)
	template.Model = "Stream"
	template.Schema = schema.New(schema.Object{
		Properties: schema.ElementMap{
			"label":   schema.String{},
			"summary": schema.String{},
			"content": schema.Object{Properties: schema.ElementMap{
				"format": schema.String{},
				"raw":    schema.String{},
				"html":   schema.String{},
			}},
		},
	})

	require.Empty(t, template.UnsupportedSchemaProperties())
}

// TestTemplate_UnsupportedSchemaProperties_OrphanTitle reproduces the real bandwagon-news
// bug: a Stream template that declares a top-level "title" property, which model.Stream
// has no accessor for.  The check must flag it.
func TestTemplate_UnsupportedSchemaProperties_OrphanTitle(t *testing.T) {

	template := NewTemplate("test-orphan-title", nil)
	template.Model = "Stream"
	template.Schema = schema.New(schema.Object{
		Properties: schema.ElementMap{
			"title":   schema.String{}, // <-- orphan: Stream has "label", not "title"
			"summary": schema.String{},
		},
	})

	require.Equal(t, []string{"title"}, template.UnsupportedSchemaProperties())
}

// TestTemplate_UnsupportedSchemaProperties_OrphanNestedType reproduces the second
// bandwagon-news bug: the "content" sub-schema declared "type" instead of "format".
// model.Content backs format/raw/html -- so "content.type" is an orphan.  This confirms
// the walk descends into nested objects and reports the dotted leaf path.
func TestTemplate_UnsupportedSchemaProperties_OrphanNestedType(t *testing.T) {

	template := NewTemplate("test-orphan-nested", nil)
	template.Model = "Stream"
	template.Schema = schema.New(schema.Object{
		Properties: schema.ElementMap{
			"content": schema.Object{Properties: schema.ElementMap{
				"type": schema.String{}, // <-- orphan: Content has "format", not "type"
				"raw":  schema.String{},
				"html": schema.String{},
			}},
		},
	})

	require.Equal(t, []string{"content.type"}, template.UnsupportedSchemaProperties())
}

// TestTemplate_NewObject_MatchesBaseSchema is the invariant that makes the whole check
// trustworthy: for every allowed model, the object returned by NewObject() must actually
// support every property declared in that model's BaseSchema().  If this ever fails, the
// NewObject/BaseSchema mapping has drifted.  UnsupportedSchemaProperties() gates on
// Model=="Stream", so this test checks the invariant DIRECTLY (schema.Get against the
// NewObject) for every model -- otherwise drift in a non-Stream mapping would slip
// through silently.
func TestTemplate_NewObject_MatchesBaseSchema(t *testing.T) {

	models := []string{
		"Stream", "None", "",
		"User", "Outbox", "Inbox", "Settings", "Conversations",
		"Domain", "Search", "SSO", "Followers", "Following", "Syndication",
		"Group", "Identity", "Rule", "Tag", "Webhook",
	}

	for _, model := range models {
		t.Run(model, func(t *testing.T) {

			template := NewTemplate("test", nil)
			template.Model = model

			baseSchema := schema.New(template.BaseSchema())
			object := template.NewObject()

			// Every property in the model's own BaseSchema must resolve to a real accessor
			// on the object NewObject() returns for the same model.
			for path := range baseSchema.AllProperties() {
				_, err := baseSchema.Get(object, path)
				require.False(t, isUnsupportedProperty(err),
					"model %q: BaseSchema property %q is not backed by NewObject()", model, path)
			}
		})
	}
}

// isUnsupportedProperty mirrors the classification used by UnsupportedSchemaProperties.
func isUnsupportedProperty(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(derp.Message(derp.RootCause(err)), "does not support this")
}
