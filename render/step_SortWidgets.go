package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StepSortWidgets represents an action-step that can edit/update Container in a streamDraft.
type StepSortWidgets struct{}

func (step StepSortWidgets) Get(renderer Renderer, buffer io.Writer) error {
	return nil
}

func (step StepSortWidgets) UseGlobalWrapper() bool {
	return true
}

func (step StepSortWidgets) Post(renderer Renderer) error {

	streamRenderer, ok := renderer.(*Stream)

	if !ok {
		return derp.NewInternalError("render.StepSortWidgets.Post", "edit-widgets can only be used on Stream data")
	}

	context := streamRenderer.context()
	request := context.Request()

	if err := request.ParseForm(); err != nil {
		return derp.Wrap(err, "render.StepSortWidgets.Post", "Error parsing form data")
	}

	stream := streamRenderer.stream
	template := streamRenderer.template()
	newWidgets := sliceof.NewObject[model.StreamWidget]()

	for _, location := range template.WidgetLocations {
		for _, value := range request.Form[location] {
			var widget model.StreamWidget

			// Move existing widgets
			if widgetID, err := primitive.ObjectIDFromHex(value); err == nil {
				if widget = stream.WidgetByID(widgetID); !widget.IsNew() {
					widget.Location = location
					newWidgets.Append(widget)
				}
				continue
			}

			// Create new widgets
			widget.StreamWidgetID = primitive.NewObjectID()
			widget.Location = location
			widget.Type = value
			widget.Label = ""

			newWidgets.Append(widget)
		}
	}

	// Apply the new data structure to the stream
	stream.Widgets = newWidgets

	// Success!
	return nil
}
