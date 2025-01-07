package build

import (
	"io"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/formdata"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepSortWidgets is a Step that can edit/update Container in a streamDraft.
type StepSortWidgets struct{}

func (step StepSortWidgets) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return nil
}

func (step StepSortWidgets) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepSortWidgets.Post"

	streamBuilder, ok := builder.(*Stream)

	if !ok {
		return Halt().WithError(derp.NewInternalError(location, "edit-widgets can only be used on Stream transaction"))
	}

	// Collect required services
	factory := streamBuilder._factory
	widgetService := factory.Widget()

	// Collect transaction from form POST
	transaction, err := formdata.Parse(builder.request())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error binding form transaction"))
	}

	// Set up some variables
	stream := streamBuilder._stream
	template := streamBuilder._template
	newWidgets := model.NewStreamWidgets()

	// Find and organize the selected widgets
	for _, widgetLocation := range template.WidgetLocations {

		widgetTypes := strings.Split(transaction.Get(widgetLocation), ",")
		for _, widgetType := range widgetTypes {
			var widget model.StreamWidget

			// Move existing widgets
			if widgetID, err := primitive.ObjectIDFromHex(widgetType); err == nil {
				if widget = stream.WidgetByID(widgetID); !widget.IsNew() {
					widget.Location = widgetLocation
					newWidgets.Append(widget)
				}
				continue
			}

			// Create new widgets
			if template.IsValidWidgetLocation(widgetLocation) {
				if widgetDefinition, ok := widgetService.Get(widgetType); ok {
					widget.StreamWidgetID = primitive.NewObjectID()
					widget.Location = widgetLocation
					widget.Type = widgetType
					widget.Label = widgetDefinition.Label

					newWidgets.Append(widget)
				}
			}
		}
	}

	// Apply the new data structure to the stream
	stream.Widgets = newWidgets

	// Success!
	return nil
}
