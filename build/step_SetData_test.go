package build

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

// TestStepSetData_Defaults verifies that "defaults" fill only the paths that are empty on the object
func TestStepSetData_Defaults(t *testing.T) {

	streamSchema := schema.New(model.StreamSchema())
	step := StepSetData{Defaults: mapof.Any{"label": "New Page"}}

	t.Run("keeps an existing value", func(t *testing.T) {
		stream := model.NewStream()
		stream.Label = "Existing Label"

		require.NoError(t, step.setDefaults(streamSchema, &stream))
		require.Equal(t, "Existing Label", stream.Label)
	})

	t.Run("fills an empty value", func(t *testing.T) {
		stream := model.NewStream()

		require.NoError(t, step.setDefaults(streamSchema, &stream))
		require.Equal(t, "New Page", stream.Label)
	})

	t.Run("fills a path inside a nil data map", func(t *testing.T) {
		stream := model.Stream{}
		dataStep := StepSetData{Defaults: mapof.Any{"data.x": "NONE"}}

		require.NoError(t, dataStep.setDefaults(streamSchema, &stream))
		require.Equal(t, "NONE", stream.Data["x"])
	})
}

// TestStepSetData_SetFormValues verifies that a posted value is written through the schema, and that a
// refused one reports the schema's validation error and never the posted form
func TestStepSetData_SetFormValues(t *testing.T) {

	streamSchema := schema.New(model.StreamSchema())
	step := StepSetData{FromForm: []string{"label"}}

	t.Run("writes a value", func(t *testing.T) {
		stream := model.NewStream()

		require.NoError(t, step.setFormValues(streamSchema, &stream, url.Values{"label": {"New Label"}}))
		require.Equal(t, "New Label", stream.Label)
	})

	t.Run("joins repeated values", func(t *testing.T) {
		stream := model.NewStream()

		require.NoError(t, step.setFormValues(streamSchema, &stream, url.Values{"label": {"a", "b"}}))
		require.Equal(t, "a,b", stream.Label)
	})

	// A schema that refuses with a validation error (enum) and with a bad request (rosetta's color format)
	refusingSchema := schema.New(schema.Object{Properties: schema.ElementMap{
		"layout": schema.String{Enum: []string{"row", "column"}},
		"color":  schema.String{Format: "color"},
	}})

	t.Run("keeps a validation error, so the form shows it", func(t *testing.T) {
		object := mapof.NewAny()
		formStep := StepSetData{FromForm: []string{"layout"}}
		err := formStep.setFormValues(refusingSchema, &object, url.Values{"layout": {"grid"}, "password": {"hunter2"}})

		require.True(t, derp.IsValidationError(err))
		require.Equal(t, "layout: Must be one of the specified values", derp.RootMessage(err))
		require.NotContains(t, fmt.Sprintf("%+v", err), "hunter2")
		require.Empty(t, object["layout"])
	})

	t.Run("reports any other refusal as a bad request", func(t *testing.T) {
		object := mapof.NewAny()
		formStep := StepSetData{FromForm: []string{"color"}}
		err := formStep.setFormValues(refusingSchema, &object, url.Values{"color": {"not-a-color"}, "password": {"hunter2"}})

		require.False(t, derp.IsValidationError(err))
		require.Equal(t, http.StatusBadRequest, derp.ErrorCode(err))
		require.NotContains(t, fmt.Sprintf("%+v", err), "hunter2")
		require.Empty(t, object["color"])
	})

	t.Run("stops at the first refusal", func(t *testing.T) {
		object := mapof.NewAny()
		formStep := StepSetData{FromForm: []string{"layout", "color"}}
		err := formStep.setFormValues(refusingSchema, &object, url.Values{"layout": {"grid"}, "color": {"#336699"}})

		require.Error(t, err)
		require.Empty(t, object["color"])
	})
}

// stubSetDataBuilder is a build.Builder exposing only the three methods that StepSetData.Post reaches
type stubSetDataBuilder struct {
	Builder
	stream *model.Stream
	req    *http.Request
}

// object implements the Builder interface, returning this stub's Stream
func (b stubSetDataBuilder) object() data.Object { return b.stream }

// request implements the Builder interface, returning this stub's request
func (b stubSetDataBuilder) request() *http.Request { return b.req }

// schema implements the Builder interface, returning the Stream schema
func (b stubSetDataBuilder) schema() schema.Schema { return schema.New(model.StreamSchema()) }

// TestStepSetData_Post_IgnoresTheQueryString confirms `from-form` reads the request body alone, so a
// value in a crafted link is never joined onto the one the visitor posted
func TestStepSetData_Post_IgnoresTheQueryString(t *testing.T) {

	body := url.Values{"label": {"Mine"}}
	request := httptest.NewRequest(http.MethodPost, "/stream/edit?label=Theirs", strings.NewReader(body.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	stream := model.NewStream()
	builder := stubSetDataBuilder{stream: &stream, req: request}

	require.Nil(t, StepSetData{FromForm: []string{"label"}}.Post(builder, nil))
	require.Equal(t, "Mine", stream.Label)
}
