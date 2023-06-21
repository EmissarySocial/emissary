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
func (step StepAsModal) Get(renderer Renderer, buffer io.Writer) ExitCondition {

	const location = "render.StepAsModal.Get"

	// Partial pages only render the modal window.  This happens MOST of the time.
	if renderer.IsPartialRequest() {

		header := renderer.context().Response().Header()
		header.Set("HX-Retarget", "aside")
		header.Set("HX-Reswap", "innerHTML")
		if step.Background == "" {
			header.Set("HX-Push", "false")
		}

		modalContent, status := step.getModalContent(renderer)

		if status.Halt {
			return ExitWithStatus(status)
		}

		if _, err := io.WriteString(buffer, modalContent); err != nil {
			return ExitError(derp.Wrap(err, location, "Error writing from builder to buffer"))
		}

		return ExitWithStatus(status).AsFullPage()
	}

	// Otherwise, we can render the modal on a page background... IF we have a background view defined.
	if step.Background == "" {
		return ExitError(derp.NewBadRequestError(location, "render.StepAsModal.Get", "Cannot open this route directly."))
	}

	// Full pages render the entire page, including the modal window
	fullPageRenderer, err := renderer.clone(step.Background)

	if err != nil {
		return ExitError(derp.Wrap(err, location, "Error creating fullPageRenderer"))
	}

	htmlTemplate := renderer.factory().Domain().Theme().HTMLTemplate
	var fullPage bytes.Buffer

	if err := htmlTemplate.ExecuteTemplate(&fullPage, "page", fullPageRenderer); err != nil {
		return ExitError(derp.Wrap(err, "render.StepAsModal.Get", "Error executing template"))
	}

	// Insert the modal into the page
	asideBegin := "<aside>"
	asideEnd := "</aside>"
	modalString, status := step.getModalContent(renderer)
	fullPageString := strings.Replace(fullPage.String(), asideBegin+asideEnd, asideBegin+modalString+asideEnd, 1)

	if _, err := io.WriteString(buffer, fullPageString); err != nil {
		return ExitError(derp.Wrap(err, location, "Error writing from builder to buffer"))
	}

	return ExitWithStatus(status).AsFullPage()
}

// Post updates the stream with approved data from the request body.
func (step StepAsModal) Post(renderer Renderer, buffer io.Writer) ExitCondition {

	// Write inner items
	status := Pipeline(step.SubSteps).Post(renderer.factory(), renderer, buffer)
	status.Error = derp.Wrap(status.Error, "render.StepAsModal.Post", "Error executing subSteps")

	return ExitWithStatus(status).WithEvent("closeModal", "true")
}

func (step StepAsModal) getModalContent(renderer Renderer) (string, PipelineStatus) {

	const location = "render.StepAsModal.getModalContent"

	// Write inner items
	var buffer bytes.Buffer

	// nolint:errcheck
	status := Pipeline(step.SubSteps).Get(renderer.factory(), renderer, &buffer)
	status.Error = derp.Wrap(status.Error, location, "Error executing subSteps")

	if status.Halt {
		return "", status
	}

	return WrapModal(renderer.context().Response(), buffer.String(), step.Options...), status
}
