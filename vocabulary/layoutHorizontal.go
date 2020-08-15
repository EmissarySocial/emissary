package vocabulary

import (
	"html"
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/schema"
)

func LayoutHorizontal(library form.Library) {

	library.Register("layout-horizontal", func(form form.Form, schema schema.Schema, value interface{}, builder *strings.Builder) error {

		var result error

		builder.WriteString(`<div class="layout-horizontal">`)
		builder.WriteString(`<div class="elements>`)

		for index, child := range form.Children {

			builder.WriteString(`<div class="element">`)
			builder.WriteString(`<div class="label">`)
			builder.WriteString(html.EscapeString(child.Label))
			builder.WriteString(`</div">`)

			if err := child.Write(library, schema, value, builder); err != nil {
				result = derp.Wrap(err, "form.widget.LayoutHorizontal", "Error rendering child", index, child)
			}
			builder.WriteString(`</div>`)
		}

		builder.WriteString(`</div>`)
		builder.WriteString(`</div>`)

		return result
	})
}
