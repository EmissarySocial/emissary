package vocabulary

import (
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"github.com/benpate/schema"
)

func LayoutHorizontal(library form.Library) {

	library.Register("layout-horizontal", func(form form.Form, schema *schema.Schema, value interface{}, b *html.Builder) error {

		var result error

		b.Div().Class("layout-horizontal")
		if form.Label != "" {
			b.Div().Class("layout-horizontal-label").InnerHTML(form.Label).Close()
		}
		b.Div().Class("layout-horizontal-elements")

		for index, child := range form.Children {

			b.Div().Class("layout-horizontal-element")

			if form.Options["show-labels"] != "false" {
				b.Div().Class("label").InnerHTML(child.Label).Close()
			}

			if err := child.Write(library, schema, value, b.SubTree()); err != nil {
				result = derp.Wrap(err, "form.widget.LayoutHorizontal", "Error rendering child", index, child)
			}

			b.Close()
		}

		b.CloseAll()

		return result
	})
}
