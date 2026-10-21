package build

import (
	"bytes"
	"io"
	"strings"
	"text/template"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepAsModal is a Step that can update the data.DataMap custom data stored in a Stream
type StepAsModal struct {
	SubSteps      []step.Step
	Options       []*template.Template
	Background    string
	BackgroundURL *template.Template
	ReplaceURL    *template.Template
	Head          []step.Step
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
			return Halt().WithError(derp.Wrap(err, location, "Writing from builder to buffer"))
		}

		result := UseResult(status).
			WithHeader("HX-Retarget", "aside").
			WithHeader("HX-Reswap", "innerHTML").
			AsFullPage()

		// The modal's own URL goes into the address bar, so it can be copied and shared.  htmx reads
		// HX-Push-Url before HX-Replace-Url, so the "false" that WrapModal wrote must go.
		if replaceURL := step.replaceURL(builder); replaceURL != "" {
			builder.response().Header().Del("HX-Push-Url")
			return result.WithHeader("HX-Replace-Url", replaceURL)
		}

		if step.Background == "" {
			result = result.WithHeader("HX-Push-Url", "false")
		}

		return result
	}

	// RULE: A page drawn behind a modal cannot draw a modal of its own, which would also loop forever
	// when two backgrounds point at one another
	if isBackgroundRequest(builder.request()) {
		return Halt().WithError(derp.BadRequest(location, "A modal cannot be the background of another modal"))
	}

	// Full pages build the entire page, including the modal window
	fullPageString, returnURL, err := step.getBackgroundPage(builder)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Building background page"))
	}

	// Insert the modal into the page
	asideBegin := "<aside>"
	asideEnd := "</aside>"
	modalString, result := step.getModalContent(builder)
	modalString = markReturnURL(modalString, returnURL)
	fullPageString = strings.Replace(fullPageString, asideBegin+asideEnd, asideBegin+modalString+asideEnd, 1)

	// Insert the modal's own <head> values, so that a shared link previews the modal and not its background
	headString, err := step.getHead(builder)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Building head"))
	}

	fullPageString = insertHead(fullPageString, headString)

	if _, err := io.WriteString(buffer, fullPageString); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Writing from builder to buffer"))
	}

	return UseResult(result).AsFullPage()
}

// Post updates the stream with approved data from the request body.
func (step StepAsModal) Post(builder Builder, buffer io.Writer) PipelineBehavior {

	// Write inner items
	result := Pipeline(step.SubSteps).Post(builder.factory(), builder, buffer)
	result.Error = derp.WrapIF(result.Error, "build.StepAsModal.Post", "Executing subSteps")

	return UseResult(result).WithEvent("closeModal", "true")
}

// getModalContent renders this step's sub-steps into the body of the modal
func (step StepAsModal) getModalContent(builder Builder) (string, PipelineResult) {

	const location = "build.StepAsModal.getModalContent"

	// Write inner items
	var buffer bytes.Buffer

	result := Pipeline(step.SubSteps).Get(builder.factory(), builder, &buffer)
	result.Error = derp.WrapIF(result.Error, location, "Executing subSteps")

	if result.Halt {
		return "", result
	}

	return WrapModal(builder.response(), buffer.String(), step.options(builder)...), result
}

// options executes each option template against the builder
func (step StepAsModal) options(builder Builder) []string {

	result := make([]string, len(step.Options))

	for index, option := range step.Options {
		result[index] = executeTemplate(option, builder)
	}

	return result
}

// replaceURL returns the local URL to write into the address bar when the modal opens, or "" for none
func (step StepAsModal) replaceURL(builder Builder) string {

	if step.ReplaceURL == nil {
		return ""
	}

	result := executeTemplate(step.ReplaceURL, builder)

	// RULE: Only a path on this server goes into the address bar
	if !isLocalPath(result) {
		return ""
	}

	return result
}

// getBackgroundPage renders the full page drawn behind the modal: the page at background-url when
// there is one and the visitor can see it, otherwise this builder's own background action.  When the
// background came from background-url, that URL is returned too, as the address to restore on close.
func (step StepAsModal) getBackgroundPage(builder Builder) (page string, returnURL string, err error) {

	const location = "build.StepAsModal.getBackgroundPage"

	if step.BackgroundURL != nil {

		page, returnURL, err := step.getBackgroundURLPage(builder)

		if err == nil {
			return page, returnURL, nil
		}

		// A visitor who cannot see the background (a private profile, say) still sees the modal,
		// over the fallback.  Anything else is a fault worth reporting.
		if !derp.IsClientError(err) {
			derp.Report(derp.Wrap(err, location, "Building background-url.  Falling back to background"))
		}
	}

	if step.Background == "" {
		return "", "", derp.BadRequest(location, "Cannot open this route directly.")
	}

	fullPageBuilder, err := builder.clone(step.Background)

	if err != nil {
		return "", "", derp.Wrap(err, location, "Creating fullPageBuilder")
	}

	page, err = renderBackground(fullPageBuilder)

	if err != nil {
		return "", "", derp.Wrap(err, location, "Rendering background")
	}

	return page, "", nil
}

// getBackgroundURLPage renders the page at this step's background-url, and returns that URL with it
func (step StepAsModal) getBackgroundURLPage(builder Builder) (string, string, error) {

	const location = "build.StepAsModal.getBackgroundURLPage"

	target := executeTemplate(step.BackgroundURL, builder)
	backgroundBuilder, err := newBackgroundBuilder(builder, target)

	if err != nil {
		return "", "", derp.Wrap(err, location, "Creating background builder", target)
	}

	page, err := renderBackground(backgroundBuilder)

	if err != nil {
		return "", "", derp.Wrap(err, location, "Rendering background", target)
	}

	return page, target, nil
}

// getHead renders this step's head steps against the modal's builder
func (step StepAsModal) getHead(builder Builder) (string, error) {

	const location = "build.StepAsModal.getHead"

	if len(step.Head) == 0 {
		return "", nil
	}

	var buffer bytes.Buffer

	if result := Pipeline(step.Head).Get(builder.factory(), builder, &buffer); result.Error != nil {
		return "", derp.Wrap(result.Error, location, "Executing head steps")
	}

	return buffer.String(), nil
}
