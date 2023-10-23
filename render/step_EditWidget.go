package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
)

// StepEditWidget represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepEditWidget struct{}

func (step StepEditWidget) Get(renderer Renderer, buffer io.Writer) PipelineBehavior {

	widget, streamWidget, _, err := step.common(renderer)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, "render.StepEditWidget.Get", "Error locating widget"))
	}

	// Render the Form
	formHTML, err := form.Editor(widget.Schema, widget.Form, streamWidget.Data, nil)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, "render.StepEditWidget.Get", "Error rendering form"))
	}

	// Wrap the form as a modal and return it to the client
	formHTML = WrapModalForm(renderer.response(), renderer.URL(), formHTML)

	// nolint:errcheck
	buffer.Write([]byte(formHTML))

	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepEditWidget) Post(renderer Renderer, _ io.Writer) PipelineBehavior {

	// Locate the widget and its configuration
	widget, streamWidget, streamRenderer, err := step.common(renderer)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, "render.StepEditWidget.Post", "Error locating widget"))
	}

	// Get the form post information
	formData := mapof.NewAny()
	if err := bind(renderer.request(), &formData); err != nil {
		return Halt().WithError(derp.Wrap(err, "render.StepEditWidget.Post", "Error binding form data"))
	}

	// Apply the form data to the widget
	f := form.New(widget.Schema, widget.Form)
	if err := f.SetAll(&streamWidget.Data, formData, nil); err != nil {
		return Halt().WithError(derp.Wrap(err, "render.StepEditWidget.Post", "Error applying form data to widget"))
	}

	// Update the stream with the new widget (in the same location)
	streamRenderer._stream.Widgets.Put(streamWidget)

	return Continue().WithEvent("closeModal", "true")
}

// common locates the widget and its configuration
func (step StepEditWidget) common(renderer Renderer) (model.Widget, model.StreamWidget, *Stream, error) {

	const location = "render.StepEditWidget.doStep"

	// WithWidget can only be used on a Stream
	streamRenderer, ok := renderer.(*Stream)

	if !ok {
		return model.Widget{}, model.StreamWidget{}, nil, derp.New(derp.CodeInternalError, location, "Renderer is not a StreamRenderer")
	}

	// User must be authenticated to view widget details
	if !streamRenderer.IsAuthenticated() {
		return model.Widget{}, model.StreamWidget{}, nil, derp.NewUnauthorizedError(location, "Anonymous user is not authorized to perform this action")
	}

	// Get the token from the request
	token := renderer.QueryParam("widgetId")

	if token == "" {
		return model.Widget{}, model.StreamWidget{}, nil, derp.NewBadRequestError(location, "Missing required parameter: widgetId")
	}

	// Try to find the widget in the stream
	streamWidget, ok := streamRenderer._stream.Widgets.Get(token)

	if !ok {
		return model.Widget{}, model.StreamWidget{}, nil, derp.NewBadRequestError(location, "Invalid widgetId", token)
	}

	widgetService := streamRenderer.factory().Widget()
	widget, ok := widgetService.Get(streamWidget.Type)

	if !ok {
		return model.Widget{}, model.StreamWidget{}, nil, derp.NewInternalError(location, "Unknown widget type", streamWidget.Type)
	}

	// TODO: LOW: This should be IsEmpty() accessor method on the Widget object
	if len(widget.Form.Children) == 0 {
		return model.Widget{}, model.StreamWidget{}, nil, derp.NewBadRequestError(location, "Widget does not support editing (empty form)", streamWidget.Type)
	}

	// TODO: LOW: This should be IsEmpty() accessor method on the Schema object
	if widget.Schema.Element == nil {
		return model.Widget{}, model.StreamWidget{}, nil, derp.NewBadRequestError(location, "Widget does not support editing (empty schema)", streamWidget.Type)
	}

	return widget, streamWidget, streamRenderer, nil
}
