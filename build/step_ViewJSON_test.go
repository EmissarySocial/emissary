package build

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/data"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/hjson/hjson-go/v4"
	"github.com/stretchr/testify/require"
)

// viewJSONBuilder is a Builder that supplies only the object and schema that "view-json" reads.
// Every other Builder method is unused, and panics if called.
type viewJSONBuilder struct {
	Builder
	stream *model.Stream
}

// schema returns the Stream's base schema
func (builder viewJSONBuilder) schema() schema.Schema {
	return schema.New(model.StreamSchema())
}

// object returns the Stream
func (builder viewJSONBuilder) object() data.Object {
	return builder.stream
}

// newViewJSONStep parses a "view-json" step from hjson, the way a Template loads it
func newViewJSONStep(t *testing.T, definition string) StepViewJSON {

	t.Helper()

	stepInfo := mapof.Any{}
	require.NoError(t, hjson.Unmarshal([]byte(definition), &stepInfo))

	parsed, err := step.NewViewJSON(stepInfo)
	require.NoError(t, err)

	return StepViewJSON(parsed)
}

// runBehavior applies a step's PipelineBehavior to a new PipelineResult
func runBehavior(behavior PipelineBehavior) PipelineResult {
	result := NewPipelineResult()
	behavior(&result)
	return result
}

// snapshot returns the JSON form of a step's rules and schema, to prove that requests never change them
func snapshot(t *testing.T, viewJSON StepViewJSON) string {
	t.Helper()
	result, err := json.Marshal([]any{viewJSON.Rules, viewJSON.Schema})
	require.NoError(t, err)
	return string(result)
}

// creditStep is the album's "artist-credit" action, reduced to a few rules
const creditStep = `{
	do: "view-json"
	content-type: "application/activity+json"
	schema: {
		type: "object"
		properties: {
			"@context": {type: "array", maxLength: 4, items: {type: "string"}}
			index: {type: "integer"}
			artist: {type: "object", properties: {id: {type: "string", format: "url"}, name: {type: "string"}}}
		}
	}
	rules: [
		{target:"", value:{type:"ArtistCredit", joinphrase:"", index:0}}
		{target:"@context.0", value:"https://www.w3.org/ns/activitystreams"}
		{target:"@context.1", value:"https://funkwhale.audio/ns"}
		{target:"id", expression:"{{.URL}}/artist-credit"}
		{target:"artist.id", path:"attributedTo.profileUrl"}
		{target:"artist.name", path:"attributedTo.name"}
	]
}`

// TestStepViewJSON_Rules covers the "rules" mode end to end: the document, its content type, and
// the shared step that every request reuses
func TestStepViewJSON_Rules(t *testing.T) {

	stream := model.NewStream()
	stream.URL = "https://band.example/album"
	stream.AttributedTo.ProfileURL = "https://band.example/@artist"
	stream.AttributedTo.Name = "Lime Bar"

	builder := viewJSONBuilder{stream: &stream}
	viewJSON := newViewJSONStep(t, creditStep)
	before := snapshot(t, viewJSON)

	t.Run("the rules write the document", func(t *testing.T) {

		// FUNKWHALE task 1.6: Funkwhale 2.0 fetches each album credit by its id, and needs
		// the empty joinphrase that a field-by-field write would drop
		var buffer bytes.Buffer
		behavior := runBehavior(viewJSON.Get(builder, &buffer))
		require.Nil(t, behavior.Error)
		require.Equal(t, "application/activity+json", behavior.ContentType)
		require.True(t, behavior.FullPage)

		result := map[string]any{}
		require.NoError(t, json.Unmarshal(buffer.Bytes(), &result))
		require.Equal(t, map[string]any{
			"@context":   []any{"https://www.w3.org/ns/activitystreams", "https://funkwhale.audio/ns"},
			"type":       "ArtistCredit",
			"id":         "https://band.example/album/artist-credit",
			"joinphrase": "",
			"index":      float64(0),
			"artist":     map[string]any{"id": "https://band.example/@artist", "name": "Lime Bar"},
		}, result)
	})

	t.Run("each request writes a new document", func(t *testing.T) {

		first, err := viewJSON.buildDocument(builder.schema(), builder.object())
		require.NoError(t, err)
		first["id"] = "changed"
		first.GetMap("artist")["name"] = "changed"

		second, err := viewJSON.buildDocument(builder.schema(), builder.object())
		require.NoError(t, err)
		require.Equal(t, "https://band.example/album/artist-credit", second["id"])
		require.Equal(t, "Lime Bar", second.GetMap("artist")["name"])
		require.Equal(t, before, snapshot(t, viewJSON), "the shared step is unchanged")
	})

	t.Run("a whole-document value replaces what came before it", func(t *testing.T) {
		ordered := newViewJSONStep(t, `{rules: [{target:"name", value:"first"}, {target:"", value:{type:"Note"}}]}`)

		result, err := ordered.buildDocument(builder.schema(), builder.object())
		require.NoError(t, err)
		require.Equal(t, mapof.Any{"type": "Note"}, result)
	})

	t.Run("a failed rule halts the request", func(t *testing.T) {
		failing := newViewJSONStep(t, `{rules: [{target:"index", value:"not a number"}], schema: {type: "object", properties: {index: {type: "integer"}}}}`)

		var buffer bytes.Buffer
		behavior := runBehavior(failing.Get(builder, &buffer))
		require.NotNil(t, behavior.Error)
		require.True(t, behavior.Halt)
		require.Empty(t, buffer.Bytes(), "no partial document is written")
	})
}

// TestStepViewJSON_Value covers the "value" mode, which now answers as application/json
func TestStepViewJSON_Value(t *testing.T) {

	viewJSON := newViewJSONStep(t, `{value: "\"hello\""}`)

	var buffer bytes.Buffer
	behavior := runBehavior(viewJSON.Get(viewJSONBuilder{}, &buffer))
	require.Nil(t, behavior.Error)
	require.Equal(t, "application/json", behavior.ContentType)
	require.Equal(t, `"hello"`, buffer.String())
}
