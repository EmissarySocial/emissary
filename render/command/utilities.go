package command

import (
	"html/template"

	"github.com/benpate/ghost/model"
	"github.com/benpate/html"
	"github.com/benpate/steranko"
)

// mustTemplate guarantees that a value is a template.Template, or else it is replaced with an empty template.
func mustTemplate(data interface{}) *template.Template {
	if t, ok := data.(*template.Template); ok {
		return t
	}

	return template.New("mising")
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

func WrapModalForm(renderer Renderer, content string) string {

	b := html.New()

	// Modal Wrapper
	b.Div().ID("modal")
	b.Div().Class("modal-underlay").Script("on click send closeModal to #modal").Close()
	b.Div().Class("modal-content")

	// Form Wrapper
	b.Form("post", "").Attr("hx-post", renderer.URL()).EndBracket()

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
