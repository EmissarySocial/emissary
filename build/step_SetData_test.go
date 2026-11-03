package build

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
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
