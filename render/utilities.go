package render

import (
	"bytes"
	"html/template"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/html"
	"github.com/benpate/steranko"
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

func WrapModalForm(renderer *Stream, content string) string {

	b := html.New()

	// Modal Wrapper
	b.Div().ID("modal")
	b.Div().Class("modal-underlay").Script("on click send closeModal to #modal").Close()
	b.Div().Class("modal-content")

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

func forwardOrTrigger(renderer *Stream, forward string, trigger string) error {

	if forward != "" {

		forward, err := executeSingleTemplate(forward, renderer)

		if err != nil {
			return derp.Wrap(err, "ghost.render.forwardOrTrigger", "Error getting template")
		}

		renderer.ctx.Response().Header().Set("HX-Redirect", forward)
		return nil
	}

	if trigger != "" {

		trigger, err := executeSingleTemplate(trigger, renderer)

		if err != nil {
			return derp.Wrap(err, "ghost.render.forwardOrTrigger", "Error getting template")
		}

		renderer.ctx.Response().Header().Set("HX-Trigger", trigger)
		return nil
	}

	return nil
}

func executeSingleTemplate(t string, renderer *Stream) (string, error) {

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
