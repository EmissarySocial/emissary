package build

import (
	"io"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepSortWidgets is a Step that can edit/update Container in a streamDraft.
type StepSortWidgets struct{}

func (step StepSortWidgets) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return nil
}

func (step StepSortWidgets) Post(builder Builder, _ io.Writer) PipelineBehavior {

	streamBuilder, ok := builder.(*Stream)

	if !ok {
		return Halt().WithError(derp.NewInternalError("build.StepSortWidgets.Post", "edit-widgets can only be used on Stream transaction"))
	}

	// Collect required services
	factory := streamBuilder._factory
	widgetService := factory.Widget()

	// Collect transaction from form POST
	transaction := mapof.NewString()

	if err := bind(builder.request(), &transaction); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepSortWidgets.Post", "Error binding form transaction"))
	}

	// Set up some variables
	stream := streamBuilder._stream
	template := streamBuilder._template
	newWidgets := model.NewStreamWidgets()

	// Find and organize the selected widgets
	for _, location := range template.WidgetLocations {

		widgetTypes := strings.Split(transaction.GetString(location), ",")
		for _, widgetType := range widgetTypes {
			var widget model.StreamWidget

			// Move existing widgets
			if widgetID, err := primitive.ObjectIDFromHex(widgetType); err == nil {
				if widget = stream.WidgetByID(widgetID); !widget.IsNew() {
					widget.Location = location
					newWidgets.Append(widget)
				}
				continue
			}

			// Create new widgets
			if template.IsValidWidgetLocation(location) {
				if widgetDefinition, ok := widgetService.Get(widgetType); ok {
					widget.StreamWidgetID = primitive.NewObjectID()
					widget.Location = location
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
