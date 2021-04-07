package vocabulary

import (
	"github.com/benpate/html"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/schema"
)

func LayoutGroup(library form.Library) {

	library.Register("layout-group", func(form form.Form, schema *schema.Schema, value interface{}, b *html.Builder) error {

		var result error

		b.Div().Class("layout-group")

		if form.Label != "" {
			b.Div().Class("layout-group-label").InnerHTML(form.Label).Close()
		}
		b.Div().Class("layout-group-elements")

		for index, child := range form.Children {

			tag := b.Div().Class("layout-group-element")

			if err := child.Write(library, schema, value, b.SubTree()); err != nil {
				result = derp.Wrap(err, "form.widget.LayoutGroup", "Error rendering child", index, child)
			}

			tag.Close()
		}

		b.CloseAll()

		return result
	})
}
