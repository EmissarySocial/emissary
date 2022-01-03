package render

import (
	"bytes"
	"html/template"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/html"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
)

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

func WrapForm(renderer Renderer, content string) string {

	b := html.New()

	// Form Wrapper
	b.Form("post", "").
		Attr("hx-post", renderer.URL()).
		Attr("hx-swap", "none").
		Attr("hx-push-url", "false").
		EndBracket()

	// Contents
	b.Grow(len(content))
	b.WriteString(content)

	// Controls
	b.Div()
	b.Input("submit", "").Class("primary").Value("Save Changes").Close()
	b.WriteString("&nbsp;")
	b.Span().Class("button").Script("on click trigger closeModal").InnerHTML("Cancel").Close()

	// Done
	b.CloseAll()

	return b.String()
}

func WrapModalForm(response *echo.Response, renderer Renderer, content string) string {
	return WrapModal(response, WrapForm(renderer, content))
}

// closeModal sets Response header to close a modal on the client and optionally forward to a new location.
func closeModal(ctx *steranko.Context, url string) {

	if url == "" {
		ctx.Response().Header().Set("HX-Trigger", `{"closeModal":"", "refreshPage": ""}`)
	} else {
		ctx.Response().Header().Set("HX-Trigger", `{"closeModal":{"nextPage":"`+url+`"}}`)
	}
}

func executeSingleTemplate(t string, renderer Renderer) (string, error) {

	executable, err := template.New("").Parse(t)

	if err != nil {
		return "", derp.Wrap(err, "ghost.render.executeSingleTemplate", "Error parsing template", t)
	}

	var buffer bytes.Buffer

	if err := executable.Execute(&buffer, renderer); err != nil {
		return "", derp.Wrap(err, "ghost.render.executeSingleTemplate", "Error executing template", t)
	}

	return buffer.String(), nil
}
