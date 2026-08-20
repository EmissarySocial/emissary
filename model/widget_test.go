package model

import (
	"os"
	"testing"

	"github.com/hjson/hjson-go/v4"
	"github.com/stretchr/testify/require"
)

// TestWidget_UnmarshalJSON confirms that a Widget definition loads its plain
// fields and its "saveSteps" pipeline together.
func TestWidget_UnmarshalJSON(t *testing.T) {

	definition := []byte(`{
		widgetId:"test"
		label:"Test Widget"
		schema:{type:"object", properties:{markdown:{type:"string", format:"markdown"}}}
		saveSteps:[
			{do:"set-data", values:{"data.html":"{{.Data.markdown | markdown}}"}}
		]
	}`)

	widget := Widget{}
	require.NoError(t, hjson.Unmarshal(definition, &widget))

	require.Equal(t, "test", widget.WidgetID)
	require.Equal(t, "Test Widget", widget.Label)
	require.NotNil(t, widget.Schema.Element)

	require.Len(t, widget.SaveSteps, 1)
	require.Equal(t, "set-data", widget.SaveSteps[0].Name())
}

// TestWidget_UnmarshalJSON_NoSaveSteps confirms that a definition without a
// "saveSteps" pipeline loads cleanly and simply carries no steps.
func TestWidget_UnmarshalJSON_NoSaveSteps(t *testing.T) {

	widget := Widget{}
	require.NoError(t, hjson.Unmarshal([]byte(`{widgetId:"test", label:"Test"}`), &widget))

	require.Equal(t, "test", widget.WidgetID)
	require.Empty(t, widget.SaveSteps)
}

// TestWidget_UnmarshalJSON_InvalidSaveStep confirms that an unrecognized step
// name in a "saveSteps" pipeline is reported instead of silently ignored.
func TestWidget_UnmarshalJSON_InvalidSaveStep(t *testing.T) {

	widget := Widget{}
	err := hjson.Unmarshal([]byte(`{widgetId:"test", saveSteps:[{do:"not-a-real-step"}]}`), &widget)

	require.Error(t, err)
}

// TestWidget_UnmarshalJSON_PreservesPopulatedFields confirms that unmarshalling
// leaves fields the Widget service populates beforehand (HTMLTemplate, Bundles)
// untouched, because the definition never names them.
func TestWidget_UnmarshalJSON_PreservesPopulatedFields(t *testing.T) {

	widget := NewWidget("test", nil)
	require.NotNil(t, widget.HTMLTemplate)
	require.NotNil(t, widget.Bundles)

	require.NoError(t, hjson.Unmarshal([]byte(`{widgetId:"test", label:"Test"}`), &widget))

	require.NotNil(t, widget.HTMLTemplate)
	require.NotNil(t, widget.Bundles)
}

// TestWidget_EmbeddedMarkdownDefinition confirms that the shipped widget-markdown
// definition declares a save pipeline, so the template and the mechanism cannot
// drift apart unnoticed.
func TestWidget_EmbeddedMarkdownDefinition(t *testing.T) {

	definition, err := os.ReadFile("../_embed/templates/widget-markdown/widget.hjson")
	require.NoError(t, err)

	widget := Widget{}
	require.NoError(t, hjson.Unmarshal(definition, &widget))

	require.Equal(t, "markdown", widget.WidgetID)
	require.Len(t, widget.SaveSteps, 1)
	require.Equal(t, "set-data", widget.SaveSteps[0].Name())
}

// TestStreamWidget_IsDataObject confirms that a StreamWidget satisfies the
// data.Object interface, which build pipelines require.
func TestStreamWidget_IsDataObject(t *testing.T) {

	streamWidget := NewStreamWidget("markdown", "Label", "TOP")

	require.Equal(t, streamWidget.StreamWidgetID.Hex(), streamWidget.ID())
	require.False(t, streamWidget.IsDeleted())

	streamWidget.SetUpdated("changed")
	require.NotZero(t, streamWidget.Updated())
	require.NotEmpty(t, streamWidget.ETag())
}
