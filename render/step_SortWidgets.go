package render

import (
	"io"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepSortWidgets represents an action-step that can edit/update Container in a streamDraft.
type StepSortWidgets struct{}

func (step StepSortWidgets) Get(renderer Renderer, buffer io.Writer) ExitCondition {
	return nil
}

func (step StepSortWidgets) Post(renderer Renderer, _ io.Writer) ExitCondition {

	streamRenderer, ok := renderer.(*Stream)

	if !ok {
		return ExitError(derp.NewInternalError("render.StepSortWidgets.Post", "edit-widgets can only be used on Stream data"))
	}

	// Collect required services
	factory := streamRenderer._factory
	context := streamRenderer._context
	widgetService := factory.Widget()

	// Collect data from form POST
	data := mapof.NewString()

	if err := context.Bind(&data); err != nil {
		return ExitError(derp.Wrap(err, "render.StepSortWidgets.Post", "Error binding form data"))
	}

	// Set up some variables
	stream := streamRenderer.stream
	template := streamRenderer.template()
	newWidgets := model.NewStreamWidgets()

	// Find and organize the selected widgets
	for _, location := range template.WidgetLocations {

		widgetTypes := strings.Split(data.GetString(location), ",")
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
