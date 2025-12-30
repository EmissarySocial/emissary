package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/formdata"
	"github.com/benpate/derp"
	"github.com/benpate/form"
)

// StepEditWidget is a Step that displays a form for editing Widgets.
type StepEditWidget struct{}

func (step StepEditWidget) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	widget, streamWidget, _, err := step.common(builder)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepEditWidget.Get", "Error locating widget"))
	}

	// Render the Form
	formHTML, err := form.Editor(widget.Schema, widget.Form, streamWidget.Data, nil)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepEditWidget.Get", "Unable to build form"))
	}

	// Wrap the form as a modal and return it to the client
	formHTML = WrapModalForm(builder.response(), builder.URL(), formHTML, widget.Form.Encoding())

	if _, err := buffer.Write([]byte(formHTML)); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepEditWidget.Get", "Error writing form HTML to buffer"))
	}

	return nil
}

// Post updates a Widget's configuration data.
func (step StepEditWidget) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepEditWidget.Post"

	// Locate the widget and its configuration
	widget, streamWidget, streamBuilder, err := step.common(builder)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error locating widget"))
	}

	// Get the form post information
	values, err := formdata.Parse(builder.request())
	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error binding form data"))
	}

	// Apply the form data to the widget
	f := form.New(widget.Schema, widget.Form)
	if err := f.SetURLValues(&streamWidget.Data, values, nil); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error applying form data to widget"))
	}

	// Update the stream with the new widget (in the same location)
	streamBuilder._stream.Widgets.Put(streamWidget)

	return Continue().WithEvent("closeModal", "true")
}

// common locates the widget and its configuration
func (step StepEditWidget) common(builder Builder) (model.Widget, model.StreamWidget, Stream, error) {

	const location = "build.StepEditWidget.doStep"

	// WithWidget can only be used on a Stream
	streamBuilder, ok := builder.(Stream)

	if !ok {
		return model.Widget{}, model.StreamWidget{}, Stream{}, derp.Internal(location, "Builder is not a StreamBuilder")
	}

	// User must be authenticated to view widget details
	if !streamBuilder.IsAuthenticated() {
		return model.Widget{}, model.StreamWidget{}, Stream{}, derp.Unauthorized(location, "Anonymous user is not authorized to perform this action")
	}

	// Get the token from the request
	token := builder.QueryParam("widgetId")

	if token == "" {
		return model.Widget{}, model.StreamWidget{}, Stream{}, derp.BadRequest(location, "Missing required parameter: widgetId")
	}

	// Try to find the widget in the stream
	streamWidget, ok := streamBuilder._stream.Widgets.Get(token)

	if !ok {
		return model.Widget{}, model.StreamWidget{}, Stream{}, derp.BadRequest(location, "Invalid widgetId", token)
	}

	widgetService := streamBuilder.factory().Widget()
	widget, ok := widgetService.Get(streamWidget.Type)

	if !ok {
		return model.Widget{}, model.StreamWidget{}, Stream{}, derp.Internal(location, "Unknown widget type", streamWidget.Type)
	}

	// TODO: LOW: This should be IsEmpty() accessor method on the Widget object
	if len(widget.Form.Children) == 0 {
		return model.Widget{}, model.StreamWidget{}, Stream{}, derp.BadRequest(location, "Widget does not support editing (empty form)", streamWidget.Type)
	}

	// TODO: LOW: This should be IsEmpty() accessor method on the Schema object
	if widget.Schema.Element == nil {
		return model.Widget{}, model.StreamWidget{}, Stream{}, derp.BadRequest(location, "Widget does not support editing (empty schema)", streamWidget.Type)
	}

	return widget, streamWidget, streamBuilder, nil
}
