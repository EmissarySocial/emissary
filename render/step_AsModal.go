package render

import (
	"bytes"
	"io"
	"strings"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepAsModal represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepAsModal struct {
	SubSteps   []step.Step
	Options    []string
	Background string
}

// Get displays a form where users can update stream data
func (step StepAsModal) Get(renderer Renderer, buffer io.Writer) PipelineBehavior {

	const location = "render.StepAsModal.Get"

	// Partial pages only render the modal window.  This happens MOST of the time.
	if renderer.IsPartialRequest() {

		modalContent, status := step.getModalContent(renderer)

		if status.Halt {
			return UseResult(status)
		}

		if _, err := io.WriteString(buffer, modalContent); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Error writing from builder to buffer"))
		}

		result := UseResult(status).
			WithHeader("HX-Retarget", "aside").
			WithHeader("HX-Reswap", "innerHTML").
			AsFullPage()

		if step.Background == "" {
			result = result.WithHeader("HX-Push-Url", "false")
		} else {
			result = result.WithHeader("HX-Push-Url", "true")
		}

		return result
	}

	// Otherwise, we can render the modal on a page background... IF we have a background view defined.
	if step.Background == "" {
		return Halt().WithError(derp.NewBadRequestError(location, "Cannot open this route directly."))
	}

	// Full pages render the entire page, including the modal window
	fullPageRenderer, err := renderer.clone(step.Background)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error creating fullPageRenderer"))
	}

	// Execute the action pipeline
	var partialPage bytes.Buffer
	factory := fullPageRenderer.factory()
	pipeline := Pipeline(fullPageRenderer.Action().Steps)

	status := pipeline.Execute(factory, fullPageRenderer, &partialPage, ActionMethodGet)

	if status.Error != nil {
		return Halt().WithError(derp.Wrap(status.Error, location, "Error executing action pipeline on fullPageRenderer"))
	}

	// Copy status values into the Response...
	status.Apply(fullPageRenderer.response())

	// Full Page requests require the theme service to wrap the rendered content
	var fullPage bytes.Buffer
	htmlTemplate := factory.Domain().Theme().HTMLTemplate
	fullPageRenderer.SetContent(partialPage.String())

	if err := htmlTemplate.ExecuteTemplate(&fullPage, "page", fullPageRenderer); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error executing template"))
	}

	// Insert the modal into the page
	asideBegin := "<aside>"
	asideEnd := "</aside>"
	modalString, result := step.getModalContent(renderer)
	fullPageString := strings.Replace(fullPage.String(), asideBegin+asideEnd, asideBegin+modalString+asideEnd, 1)

	if _, err := io.WriteString(buffer, fullPageString); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error writing from builder to buffer"))
	}

	return UseResult(result).AsFullPage()
}

// Post updates the stream with approved data from the request body.
func (step StepAsModal) Post(renderer Renderer, buffer io.Writer) PipelineBehavior {

	// Write inner items
	result := Pipeline(step.SubSteps).Post(renderer.factory(), renderer, buffer)
	result.Error = derp.Wrap(result.Error, "render.StepAsModal.Post", "Error executing subSteps")

	return UseResult(result).WithEvent("closeModal", "true")
}

func (step StepAsModal) getModalContent(renderer Renderer) (string, PipelineResult) {

	const location = "render.StepAsModal.getModalContent"

	// Write inner items
	var buffer bytes.Buffer

	result := Pipeline(step.SubSteps).Get(renderer.factory(), renderer, &buffer)
	result.Error = derp.Wrap(result.Error, location, "Error executing subSteps")

	if result.Halt {
		return "", result
	}

	return WrapModal(renderer.response(), buffer.String(), step.Options...), result
}
