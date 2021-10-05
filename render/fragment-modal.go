package render

import "github.com/benpate/html"

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
