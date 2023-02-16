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
func (step StepAsModal) Get(renderer Renderer, buffer io.Writer) error {

	const location = "render.StepAsModal.Get"

	// Partial pages only render the modal window.  This happens MOST of the time.
	if renderer.IsPartialRequest() {

		header := renderer.context().Response().Header()
		header.Set("HX-Retarget", "aside")
		header.Set("HX-Reswap", "innerHTML")
		if step.Background == "" {
			header.Set("HX-Push", "false")
		}

		if _, err := io.WriteString(buffer, step.getModalContent(renderer)); err != nil {
			return derp.Wrap(err, location, "Error writing from builder to buffer")
		}

		return nil
	}

	if step.Background == "" {
		return derp.NewBadRequestError(location, "render.StepAsModal.Get", "Cannot open this route directly.")
	}

	// Full pages render the entire page, including the modal window
	fullPageRenderer, err := NewRenderer(renderer.factory(), renderer.context(), renderer.object(), step.Background)

	if err != nil {
		return derp.Wrap(err, location, "Error creating fullPageRenderer")
	}

	htmlTemplate := renderer.factory().Domain().Theme().HTMLTemplate
	var fullPage bytes.Buffer

	if err := htmlTemplate.ExecuteTemplate(&fullPage, "page", fullPageRenderer); err != nil {
		return derp.Wrap(err, "render.StepAsModal.Get", "Error executing template")
	}

	// Insert the modal into the page
	asideBegin := "<aside>"
	asideEnd := "</aside>"
	modalString := step.getModalContent(renderer)
	fullPageString := strings.Replace(fullPage.String(), asideBegin+asideEnd, asideBegin+modalString+asideEnd, 1)

	if _, err := io.WriteString(buffer, fullPageString); err != nil {
		return derp.Wrap(err, location, "Error writing from builder to buffer")
	}

	return nil
}

func (step StepAsModal) UseGlobalWrapper() bool {
	return false
}

// Post updates the stream with approved data from the request body.
func (step StepAsModal) Post(renderer Renderer) error {

	// Write inner items
	if err := Pipeline(step.SubSteps).Post(renderer.factory(), renderer); err != nil {
		return derp.Wrap(err, "render.StepAsModal.Post", "Error executing subSteps")
	}

	CloseModal(renderer.context(), "")
	return nil
}

func (step StepAsModal) getModalContent(renderer Renderer) string {

	const location = "render.StepAsModal.getModalContent"

	// Write inner items
	var buffer bytes.Buffer
	if err := Pipeline(step.SubSteps).Get(renderer.factory(), renderer, &buffer); err != nil {
		derp.Report(derp.Wrap(err, location, "Error executing subSteps"))
		return ""
	}

	return WrapModal(renderer.context().Response(), buffer.String(), step.Options...)

}
