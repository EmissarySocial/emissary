package service

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

// TestWidgetAdd_RejectsUnknownSchemaFormat pushes a widget definition with a typo'd
// schema format through the real load path and confirms that Add rejects it at load
// time, while an otherwise-identical definition with a valid format loads cleanly.
func TestWidgetAdd_RejectsUnknownSchemaFormat(t *testing.T) {

	filesystem := fstest.MapFS{
		"widget.html": &fstest.MapFile{Data: []byte(`<div>WIDGET</div>`)},
	}

	valid := []byte(`{
		label: Test Widget
		schema: {
			type: object
			properties: {
				link: {
					type: string
					format: url
				}
			}
		}
	}`)

	// Identical except the format name is misspelled ("ur1" instead of "url")
	invalid := []byte(`{
		label: Test Widget
		schema: {
			type: object
			properties: {
				link: {
					type: string
					format: ur1
				}
			}
		}
	}`)

	widgetService := NewWidget(nil)

	require.NoError(t, widgetService.Add("valid-widget", filesystem, valid))

	err := widgetService.Add("invalid-widget", filesystem, invalid)
	require.Error(t, err)
	require.ErrorContains(t, err, "unrecognized format name")
}
