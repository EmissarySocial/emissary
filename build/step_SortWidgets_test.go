package build

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// stubWidgetFactory satisfies the Factory interface by embedding it (nil) and overriding only
// Widget(), which is the single service StepSortWidgets reaches for.  Any other Factory call
// would panic, which is correct: this step makes none.
type stubWidgetFactory struct {
	Factory
	widgetService *service.Widget
}

// Widget returns the stubbed widget library. Implements the Factory interface.
func (factory stubWidgetFactory) Widget() *service.Widget {
	return factory.widgetService
}

// newSortWidgetsBuilder assembles the smallest Stream builder that StepSortWidgets can run
// against: a widget library holding one definition, a template with two valid locations, and a
// POST carrying the editor's form values.
func newSortWidgetsBuilder(t *testing.T, stream *model.Stream, values url.Values) Stream {
	t.Helper()

	widgetService := service.NewWidget(nil)
	require.Nil(t, widgetService.Add("markdown", fstest.MapFS{}, []byte(`{widgetId:"markdown", label:"Markdown"}`)))

	request := httptest.NewRequest(http.MethodPost, "/000000000000000000000000/widgets", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	template := model.NewTemplate("test", nil)
	template.WidgetLocations = []string{"TOP", "BOTTOM"}

	return Stream{
		_stream: stream,
		CommonWithTemplate: CommonWithTemplate{
			_template: template,
			Common: Common{
				_factory: stubWidgetFactory{widgetService: &widgetService},
				_request: request,
			},
		},
	}
}

// applyBehavior runs a step's PipelineBehavior and returns the PipelineResult it produced
func applyBehavior(behavior PipelineBehavior) PipelineResult {
	result := NewPipelineResult()
	if behavior != nil {
		behavior(&result)
	}
	return result
}

// TestStepSortWidgets_ReswapOnlyForNewWidgets covers the difference between rearranging widgets
// and creating one.
//
// The editor posts through a form that swaps all of "main", so every save rebuilds the canvas.
// That is destructive during a drag -- Sortable's state and its ghost go with the old DOM -- and
// it interrupts the width animation a layout control just started.  A rearrangement needs none
// of it: the browser is already showing what was saved.
//
// A newly created widget is the one case that does: the browser is still showing the chip it
// cloned out of the tray, which carries the widget TYPE where its permanent ID belongs.  Left in
// place, the next save reads that type again and creates a second copy.
func TestStepSortWidgets_ReswapOnlyForNewWidgets(t *testing.T) {

	existingID := primitive.NewObjectID()

	newStreamWithWidget := func() *model.Stream {
		stream := model.NewStream()
		stream.Widgets = model.NewStreamWidgets()
		stream.Widgets.Append(model.StreamWidget{
			StreamWidgetID: existingID,
			Type:           "markdown",
			Location:       "TOP",
			Label:          "Markdown",
		})
		return &stream
	}

	t.Run("moving an existing widget leaves the page alone", func(t *testing.T) {

		stream := newStreamWithWidget()
		builder := newSortWidgetsBuilder(t, stream, url.Values{
			"TOP":    {""},
			"BOTTOM": {existingID.Hex()},
		})

		result := applyBehavior(StepSortWidgets{}.Post(builder, io.Discard))

		require.Equal(t, "none", result.Headers["HX-Reswap"])
		require.Equal(t, 1, len(stream.Widgets), "the widget must be moved, not duplicated")
		require.Equal(t, "BOTTOM", stream.Widgets[0].Location)
		require.Equal(t, existingID, stream.Widgets[0].StreamWidgetID, "an existing widget keeps its ID")
	})

	t.Run("creating a widget redraws the page", func(t *testing.T) {

		stream := newStreamWithWidget()
		builder := newSortWidgetsBuilder(t, stream, url.Values{
			"TOP":    {existingID.Hex()},
			"BOTTOM": {"markdown"},
		})

		result := applyBehavior(StepSortWidgets{}.Post(builder, io.Discard))

		require.NotContains(t, result.Headers, "HX-Reswap", "the new widget needs its server-assigned ID")
		require.Equal(t, 2, len(stream.Widgets))
		require.False(t, stream.Widgets[1].StreamWidgetID.IsZero(), "the new widget must be given a real ID")
		require.NotEqual(t, existingID, stream.Widgets[1].StreamWidgetID)
	})

	t.Run("a layout-control save touches no widget and leaves the page alone", func(t *testing.T) {

		stream := newStreamWithWidget()
		builder := newSortWidgetsBuilder(t, stream, url.Values{
			"TOP":        {existingID.Hex()},
			"BOTTOM":     {""},
			"data.width": {"MEDIUM"},
		})

		result := applyBehavior(StepSortWidgets{}.Post(builder, io.Discard))

		require.Equal(t, "none", result.Headers["HX-Reswap"])
		require.Equal(t, 1, len(stream.Widgets))
	})
}
