package build

import (
	"bytes"
	"io"
	"strings"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepAsModal is a Step that can update the data.DataMap custom data stored in a Stream
type StepAsModal struct {
	SubSteps   []step.Step
	Options    []string
	Background string
}

// Get displays a form where users can update stream data
func (step StepAsModal) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepAsModal.Get"

	// Partial pages only build the modal window.  This happens MOST of the time.
	if builder.IsPartialRequest() {

		modalContent, status := step.getModalContent(builder)

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
		}

		return result
	}

	// Otherwise, we can build the modal on a page background... IF we have a background view defined.
	if step.Background == "" {
		return Halt().WithError(derp.BadRequestError(location, "Cannot open this route directly."))
	}

	// Full pages build the entire page, including the modal window
	fullPageBuilder, err := builder.clone(step.Background)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to create fullPageBuilder"))
	}

	// Execute the action pipeline
	var partialPage bytes.Buffer
	factory := fullPageBuilder.factory()
	pipeline := Pipeline(fullPageBuilder.action().Steps)

	status := pipeline.Execute(factory, fullPageBuilder, &partialPage, ActionMethodGet)

	if status.Error != nil {
		return Halt().WithError(derp.Wrap(status.Error, location, "Error building modal with fullPageBuilder"))
	}

	// Copy status values into the Response...
	status.Apply(fullPageBuilder.response())

	// Full Page requests require the theme service to wrap the built content
	var fullPage bytes.Buffer
	htmlTemplate := factory.Domain().Theme().HTMLTemplate
	fullPageBuilder.SetContent(partialPage.String())

	if err := htmlTemplate.ExecuteTemplate(&fullPage, "page", fullPageBuilder); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error executing template"))
	}

	// Insert the modal into the page
	asideBegin := "<aside>"
	asideEnd := "</aside>"
	modalString, result := step.getModalContent(builder)
	fullPageString := strings.Replace(fullPage.String(), asideBegin+asideEnd, asideBegin+modalString+asideEnd, 1)

	if _, err := io.WriteString(buffer, fullPageString); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error writing from builder to buffer"))
	}

	return UseResult(result).AsFullPage()
}

// Post updates the stream with approved data from the request body.
func (step StepAsModal) Post(builder Builder, buffer io.Writer) PipelineBehavior {

	// Write inner items
	result := Pipeline(step.SubSteps).Post(builder.factory(), builder, buffer)
	result.Error = derp.WrapIF(result.Error, "build.StepAsModal.Post", "Error executing subSteps")

	return UseResult(result).WithEvent("closeModal", "true")
}

func (step StepAsModal) getModalContent(builder Builder) (string, PipelineResult) {

	const location = "build.StepAsModal.getModalContent"

	// Write inner items
	var buffer bytes.Buffer

	result := Pipeline(step.SubSteps).Get(builder.factory(), builder, &buffer)
	result.Error = derp.WrapIF(result.Error, location, "Error executing subSteps")

	if result.Halt {
		return "", result
	}

	return WrapModal(builder.response(), buffer.String(), step.Options...), result
}
