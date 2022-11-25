package render

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/maps"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
)

// WrapInlineSuccess sends a confirmation message to the #inline-confirmation element
func WrapInlineSuccess(ctx echo.Context, message any) error {

	ctx.Response().Header().Set("HX-Reswap", "innerHTML")
	ctx.Response().Header().Set("HX-Retarget", "#htmx-response-message")

	return ctx.HTML(http.StatusOK, `<span class="green">`+convert.String(message)+`</span>`)
}

// WrapInlineError sends a confirmation message to the #inline-confirmation element
func WrapInlineError(ctx echo.Context, err error) error {

	ctx.Response().Header().Set("HX-Reswap", "innerHTML")
	ctx.Response().Header().Set("HX-Retarget", "#htmx-response-message")

	if derpError, ok := err.(derp.SingleError); ok {
		derp.Report(derpError)
		return ctx.HTML(http.StatusOK, `<span class="red">`+derpError.Message+`</span>`)
	}

	derp.Report(err)
	return ctx.HTML(http.StatusOK, `<span class="red">`+derp.Message(err)+`</span>`)
}

func WrapModal(response *echo.Response, content string, options ...string) string {

	// These two headers make it a modal
	header := response.Header()
	header.Set("HX-Retarget", "aside")
	header.Set("HX-Push", "false")

	// Build the HTML
	b := html.New()

	// Modal Wrapper
	b.Div().ID("modal").Script("install Modal").Data("hx-swap", "none")
	b.Div().ID("modal-underlay").Close()
	b.Div().ID("modal-window").EndBracket() // this is needed because we're embedding foreign content below.

	// Contents
	b.Grow(len(content))
	b.WriteString(content)

	// Done
	b.CloseAll()

	return b.String()
}

func WrapModalWithCloseButton(response *echo.Response, content string, options ...string) string {
	b := html.New()

	b.Div()
	b.Button().Script("on click trigger closeModal").InnerHTML("Close Window")

	return WrapModal(response, content+b.String())
}

func WrapForm(endpoint string, content string, options ...string) string {

	optionMap := parseOptions(options...)

	b := html.New()

	// Form Wrapper
	b.Form("post", "").
		Attr("hx-post", endpoint).
		Attr("hx-swap", "none").
		Attr("hx-push-url", "false").
		Script("init send checkFormRules(changed:me as Values)").
		EndBracket()

	// Contents
	b.Grow(len(content))
	b.WriteString(content)

	// Controls
	b.Div()
	submitLabel := first.String(optionMap.GetString("submit-label"), "Save Changes")
	savingLabel := first.String(optionMap.GetString("saving-label"), "Saving...")
	b.Button().Type("submit").Class("htmx-request-hide primary").InnerHTML(submitLabel).Close()
	b.Button().Type("button").Class("htmx-request-show primary").Attr("disabled", "true").InnerHTML(savingLabel).Close()

	if optionMap.GetString("cancel-button") != "hide" {
		cancelLabel := first.String(optionMap.GetString("cancel-label"), "Cancel")
		b.Space()
		b.Button().Type("button").Script("on click trigger closeModal").InnerHTML(cancelLabel).Close()
		b.Space()
	}

	b.Span().ID("htmx-response-message").Close()

	// Done
	b.CloseAll()

	return b.String()
}

func WrapModalForm(response *echo.Response, endpoint string, content string, options ...string) string {
	return WrapModal(response, WrapForm(endpoint, content, options...), options...)
}

// CloseModal sets Response header to close a modal on the client and optionally forward to a new location.
func CloseModal(ctx echo.Context, url string) {

	if url == "" {
		ctx.Response().Header().Set("HX-Trigger", `{"closeModal":"", "refreshPage": ""}`)
	} else {
		ctx.Response().Header().Set("HX-Trigger", `closeModal`)
		ctx.Response().Header().Set("HX-Redirect", url)
	}
}

func RefreshPage(ctx echo.Context) {
	header := ctx.Response().Header()
	header.Set("HX-Trigger", "refreshPage")
	header.Set("HX-Reswap", "none")
}

// getAuthorization extracts a model.Authorization record from the steranko.Context
func getAuthorization(ctx *steranko.Context) model.Authorization {

	if claims, err := ctx.Authorization(); err == nil {

		if auth, ok := claims.(*model.Authorization); ok {
			return *auth
		}
	}

	return model.NewAuthorization()
}

// useGlobalWrapper returns TRUE if all steps can use the global wrapper
// if any cannot, then it returns false.
func useGlobalWrapper(steps []step.Step) bool {

	for _, item := range steps {
		if !ExecutableStep(item).UseGlobalWrapper() {
			return false
		}
	}

	return true
}

// templateLike bridges the interfaces of html/template and text/template so that
// either one can be used
type templateLike interface {
	Execute(io.Writer, any) error
}

// execTemplate provides a simplified interface for executing known/trusted
// templaes. If there is an error during execution, it is reported via
// derp.Report, but this function does not halt, instead returning an empty string
func execTemplate(template templateLike, data any) string {

	var buffer bytes.Buffer

	if err := template.Execute(&buffer, data); err != nil {
		derp.Report(err)
		return ""
	}

	return buffer.String()
}

// finalizeAddStream takes all of the follow-on actions required to initialize a new stream.
// - sets the author to the current user
// - executes the correct "init" action for this template
// - saves the stream (if not already saved by "init")
// - executes any additional "with-stream" steps
func finalizeAddStream(factory Factory, context *steranko.Context, stream *model.Stream, template *model.Template, pipeline Pipeline) error {

	const location = "render.finalizeAddStream"

	// Create stream renderer
	renderer, err := NewStream(factory, context, template, stream, "view")

	if err != nil {
		return derp.Wrap(err, location, "Error creating renderer", stream)
	}

	// Assign the current user as the author (with silent failure)
	user, _ := renderer.getUser()
	renderer.stream.SetAuthor(user)

	// TODO: MEDIUM: Set Streamort order??

	// If there is an "init" step for the stream's template, then execute it now
	if action := template.Action("init"); action != nil {
		if err := Pipeline(action.Steps).Post(factory, &renderer); err != nil {
			return derp.Wrap(err, location, "Unable to execute 'init' action on stream")
		}
	}

	// Execute additional "with-stream" steps
	if !pipeline.IsEmpty() {
		if err := pipeline.Post(factory, &renderer); err != nil {
			return derp.Wrap(err, location, "Unable to execute action steps on stream")
		}
	}

	return nil
}

// parseOptions parses a string of options into a map of key/value pairs
func parseOptions(options ...string) maps.Map {

	result := maps.New()

	for _, item := range options {
		parts := strings.Split(item, ":")
		result.SetString(parts[0], parts[1])
	}

	return result
}
