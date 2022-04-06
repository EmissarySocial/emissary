package render

import (
	"bytes"
	"html/template"
	"io"

	"github.com/benpate/derp"
	"github.com/benpate/html"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/model"
)

func WrapModal(response *echo.Response, content string) string {

	// These two headers make it a modal
	header := response.Header()
	header.Set("HX-Retarget", "aside")
	header.Set("HX-Push", "false")

	// Build the HTML
	b := html.New()

	// Modal Wrapper
	b.Div().ID("modal").Script("install Modal").EndBracket()
	b.Div().Class("modal-underlay").Close()
	b.Div().Class("modal-content").EndBracket() // this is needed because we're embedding foreign content below.

	// Contents
	b.Grow(len(content))
	b.WriteString(content)

	// Done
	b.CloseAll()

	return b.String()
}

func WrapModalWithCloseButton(response *echo.Response, content string) string {
	b := html.New()

	b.Div()
	b.Button().Script("on click trigger closeModal").InnerHTML("Close Window")

	return WrapModal(response, content+b.String())
}

func WrapForm(endpoint string, content string) string {

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
	b.Button().Type("submit").Class("primary").InnerHTML("Save Changes").Close()
	b.Space()
	b.Button().Type("button").Script("on click trigger closeModal").InnerHTML("Cancel").Close()

	// Done
	b.CloseAll()

	return b.String()
}

func WrapModalForm(response *echo.Response, endpoint string, content string) string {
	return WrapModal(response, WrapForm(endpoint, content))
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

func executeSingleTemplate(t string, renderer Renderer) (string, error) {

	executable, err := template.New("").Parse(t)

	if err != nil {
		return "", derp.Wrap(err, "whisper.render.executeSingleTemplate", "Error parsing template", t)
	}

	var buffer bytes.Buffer

	if err := executable.Execute(&buffer, renderer); err != nil {
		return "", derp.Wrap(err, "whisper.render.executeSingleTemplate", "Error executing template", t)
	}

	return buffer.String(), nil
}

// getAuthorization extracts a model.Authorization record from the steranko.Context
func getAuthorization(ctx *steranko.Context) *model.Authorization {

	if claims, err := ctx.Authorization(); err == nil {

		if auth, ok := claims.(*model.Authorization); ok {
			return auth
		}
	}

	result := model.NewAuthorization()
	return &result
}

// finalizeAddStream takes all of the follow-on actions required to initialize a new stream.
// - sets the author to the current user
// - executes the correct "init" action for this template
// - saves the stream (if not already saved by "init")
// - executes any additional "with-stream" steps
func finalizeAddStream(buffer io.Writer, factory Factory, context *steranko.Context, stream *model.Stream, template *model.Template, pipeline Pipeline) error {

	const location = "render.finalizeAddStream"

	// Create stream renderer
	action := template.Action("view")
	renderer, err := NewStream(factory, context, template, action, stream)

	if err != nil {
		return derp.Wrap(err, location, "Error creating renderer", stream)
	}

	// Assign the current user as the author (with silent failure)
	renderer.setAuthor()

	// TODO: Sort order??

	// If there is an "init" step for the stream's template, then execute it now
	if action := template.Action("init"); action != nil {
		if err := Pipeline(action.Steps).Post(factory, &renderer, buffer); err != nil {
			return derp.Wrap(err, location, "Unable to execute 'init' action on stream")
		}
	}

	/*/ If the stream was not saved by the "init" steps, then save it now
	if stream.IsNew() {

		streamService := factory.Stream()
		if err := streamService.Save(stream, "Created"); err != nil {
			return derp.Wrap(err, location, "Error saving stream stream to database")
		}
	}*/

	// Execute additional "with-stream" steps
	if !pipeline.IsEmpty() {
		if err := pipeline.Post(factory, &renderer, buffer); err != nil {
			return derp.Wrap(err, location, "Unable to execute action steps on stream")
		}
	}

	return nil
}
