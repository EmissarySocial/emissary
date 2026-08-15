package build

import (
	"io"

	"github.com/benpate/derp"
)

// executeWidgetSaveSteps runs the "saveSteps" pipeline that each Widget definition declares,
// against every matching Widget in the Stream being saved.  This lets a Widget derive
// stored values once -- for example, rendering HTML from Markdown source -- instead of
// recomputing them on every page view.  Widgets that declare no pipeline cost nothing.
func executeWidgetSaveSteps(builder Builder, buffer io.Writer, actionMethod ActionMethod) error {

	const location = "build.executeWidgetSaveSteps"

	// RULE: Only Streams contain Widgets, so every other object type is a no-op
	streamBuilder, ok := builder.(Stream)

	if !ok {
		return nil
	}

	stream := streamBuilder._stream
	widgetService := streamBuilder.factory().Widget()

	for index := range stream.Widgets {

		// Address the Widget in place, so that the pipeline's changes are kept
		streamWidget := &stream.Widgets[index]

		// An unrecognized Widget type has no definition, and so no pipeline
		definition, ok := widgetService.Get(streamWidget.Type)

		if !ok {
			continue
		}

		if len(definition.SaveSteps) == 0 {
			continue
		}

		// Inject the definition, which the Widget builder's schema is drawn from
		streamWidget.Widget = definition

		// Execute the Widget's pipeline against the Widget itself
		widgetBuilder := NewWidget(&streamBuilder, streamWidget)

		if result := Pipeline(definition.SaveSteps).Execute(streamBuilder.factory(), widgetBuilder, buffer, actionMethod); result.Error != nil {
			return derp.Wrap(result.Error, location, "Error executing save pipeline for widget", streamWidget.Type)
		}
	}

	// Every widget has had its moment in the sun.
	return nil
}
