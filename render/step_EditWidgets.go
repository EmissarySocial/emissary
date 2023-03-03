package render

import (
	"io"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
)

// StepEditWidgets represents an action-step that can edit/update Container in a streamDraft.
type StepEditWidgets struct {
	Filename string
}

func (step StepEditWidgets) Get(renderer Renderer, buffer io.Writer) error {
	if err := renderer.executeTemplate(buffer, step.Filename, renderer); err != nil {
		return derp.Wrap(err, "render.StepEditWidgets.Get", "Error executing template")
	}

	return nil
}

func (step StepEditWidgets) UseGlobalWrapper() bool {
	return true
}

func (step StepEditWidgets) Post(renderer Renderer) error {

	streamRenderer, ok := renderer.(*Stream)

	if !ok {
		return derp.NewInternalError("render.StepEditWidgets.Post", "edit-widgets can only be used on Stream data")
	}

	// Collect Form data
	request := streamRenderer.context().Request()
	if err := request.ParseForm(); err != nil {
		return derp.Wrap(err, "render.StepEditWidgets.Post", "Error parsing form data")
	}

	// Other required values
	template := streamRenderer.template()
	stream := streamRenderer.stream

	// Clear existing widgets and set new values.
	stream.Widgets = mapof.NewObject[sliceof.String]()

	for _, location := range template.WidgetLocations {
		locations := convert.SliceOfString(request.Form["widgets."+location])
		stream.Widgets[location] = locations
	}

	// Success!
	return nil
}
