package model

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/hjson/hjson-go/v4"
	"github.com/stretchr/testify/require"
)

// TestActivityStreamSchema_EveryArrayHasAMaxLength requires a limit on every list in the base
// schema, because rosetta bounds a list's length only through the schema
func TestActivityStreamSchema_EveryArrayHasAMaxLength(t *testing.T) {

	base := ActivityStreamSchema()

	for name, element := range base.Properties {
		if array, isArray := element.(schema.Array); isArray {
			require.Positive(t, array.MaxLength, "%s has no MaxLength", name)
		}
	}

	require.Equal(t, schema.Any{}, base.Wildcard, "every other property stays allowed")
}

// TestTemplate_SocialTargetSchema covers a Template with no socialSchema, one that adds
// properties, and one that replaces a base property and the wildcard
func TestTemplate_SocialTargetSchema(t *testing.T) {

	t.Run("no socialSchema writes into the base schema", func(t *testing.T) {
		template := NewTemplate("test", nil)
		require.Equal(t, schema.New(ActivityStreamSchema()), template.SocialTargetSchema())
	})

	t.Run("socialSchema properties are added to the base", func(t *testing.T) {
		template := NewTemplate("test", nil)
		template.SocialSchema = schema.New(schema.Object{Properties: schema.ElementMap{
			"artists": schema.Array{Items: schema.Any{}, MaxLength: 16},
		}})

		target, isObject := template.SocialTargetSchema().Element.(schema.Object)
		require.True(t, isObject)
		require.Equal(t, schema.Array{Items: schema.Any{}, MaxLength: 16}, target.Properties["artists"])
		require.Equal(t, ActivityStreamSchema().Properties["@context"], target.Properties["@context"])
		require.Equal(t, schema.Any{}, target.Wildcard)
	})

	t.Run("socialSchema replaces a base property and the wildcard", func(t *testing.T) {
		template := NewTemplate("test", nil)
		template.SocialSchema = schema.New(schema.Object{
			Properties: schema.ElementMap{"tag": schema.String{}},
			Wildcard:   schema.String{},
		})

		target, isObject := template.SocialTargetSchema().Element.(schema.Object)
		require.True(t, isObject)
		require.Equal(t, schema.String{}, target.Properties["tag"])
		require.Equal(t, schema.String{}, target.Wildcard)
	})

	t.Run("each call returns a new schema", func(t *testing.T) {
		template := NewTemplate("test", nil)
		template.SocialSchema = schema.New(schema.Object{Properties: schema.ElementMap{"a": schema.String{}}})

		first, _ := template.SocialTargetSchema().Element.(schema.Object)
		first.Properties["b"] = schema.String{}

		second, _ := template.SocialTargetSchema().Element.(schema.Object)
		require.NotContains(t, second.Properties, "b")
		require.NotContains(t, ActivityStreamSchema().Properties, "a")
	})
}

// TestTemplate_SocialSchemaIsValid covers a missing socialSchema, an object, and a scalar
func TestTemplate_SocialSchemaIsValid(t *testing.T) {

	template := NewTemplate("test", nil)
	require.True(t, template.SocialSchemaIsValid(), "no socialSchema")

	template.SocialSchema = schema.New(schema.Object{})
	require.True(t, template.SocialSchemaIsValid(), "an object")

	template.SocialSchema = schema.New(schema.String{})
	require.False(t, template.SocialSchemaIsValid(), "a string")
}

// TestTemplate_Inherit_SocialSchema requires a child to inherit its parent's socialSchema
// properties, and to keep its own
func TestTemplate_Inherit_SocialSchema(t *testing.T) {

	parent := NewTemplate("parent", nil)
	parent.SocialSchema = schema.New(schema.Object{Properties: schema.ElementMap{"a": schema.String{}}})

	child := NewTemplate("child", nil)
	child.SocialSchema = schema.New(schema.Object{Properties: schema.ElementMap{"b": schema.String{}}})
	child.Inherit(&parent)

	properties := child.SocialSchema.Element.(schema.Object).Properties
	require.Equal(t, schema.ElementMap{"a": schema.String{}, "b": schema.String{}}, properties)

	orphan := NewTemplate("orphan", nil)
	orphan.Inherit(&parent)
	require.Equal(t, parent.SocialSchema, orphan.SocialSchema, "a child with no socialSchema takes the parent's")
}

// TestTemplate_SocialRules_TypedSchemas runs social rules the way Stream.JSONLD does: reading
// the Stream through the Template's schema, and writing through SocialTargetSchema
func TestTemplate_SocialRules_TypedSchemas(t *testing.T) {

	template := NewTemplate("test", nil)
	err := hjson.Unmarshal([]byte(`{
		schema: {
			type: object
			properties: {
				data: {
					type: object
					properties: {
						links: {type: "object", wildcard: {type: "string", format: "url"}}
					}
				}
			}
		}
		socialRules: [
			{target:"@context", append:"https://funkwhale.audio/ns"}
			{target:"links", forEach:"data.links", rules:[
				{target:"label", path:"key"}
				{target:"href", path:"value"}
			]}
			{target:"artists.0.type", value:"Artist"}
			{target:"artists.0.id", path:"attributedTo.profileUrl"}
			{target:"position", path:"rank"}
			{target:"url", value:{}}
			{target:"url.type", value:"Link"}
			{target:"url.href", path:"url"}
		]
		socialSchema: {
			type: object
			properties: {
				artists: {
					type: array
					maxLength: 1
					items: {type: "object", properties: {type: {type: "string"}, id: {type: "string"}}}
				}
				url: {type: "object", properties: {type: {type: "string"}, href: {type: "string"}}}
			}
		}
	}`), &template)
	require.NoError(t, err)
	require.True(t, template.SocialSchemaIsValid())

	// Template.Add gives every template schema its model's base schema
	template.Schema.Inherit(schema.New(template.BaseSchema()))

	stream := NewStream()
	stream.URL = "https://band.example/song"
	stream.Rank = 3
	stream.AttributedTo.ProfileURL = "https://band.example/@artist"
	stream.Data = mapof.Any{"links": mapof.Any{"SPOTIFY": "https://spotify.example/a"}}

	result := mapof.Any{
		"@context": sliceof.Any{"https://www.w3.org/ns/activitystreams"},
		"url":      "https://band.example/song",
	}

	t.Run("each rule reads and writes through its schema", func(t *testing.T) {
		require.NoError(t, template.SocialRules.Execute(template.Schema, &stream, template.SocialTargetSchema(), &result))

		require.Equal(t, &sliceof.Any{"https://www.w3.org/ns/activitystreams", "https://funkwhale.audio/ns"}, result["@context"])
		require.Equal(t, &sliceof.Object[mapof.Any]{{"label": "SPOTIFY", "href": "https://spotify.example/a"}}, result["links"])
		require.Equal(t, &sliceof.Any{mapof.Any{"type": "Artist", "id": "https://band.example/@artist"}}, result["artists"])
		require.Equal(t, 3, result["position"])
		require.Equal(t, mapof.Any{"type": "Link", "href": "https://band.example/song"}, result["url"])
	})

	t.Run("an index past the array's maxLength is refused", func(t *testing.T) {

		// Beneath an Any, index 1 of a one-item list would be accepted, so only maxLength refuses it
		err := template.SocialTargetSchema().Set(&result, "artists.1.id", "https://elsewhere.example")
		require.Error(t, err)
		require.Len(t, *result["artists"].(*sliceof.Any), 1)
	})
}
