package vocabulary

import (
	"html"
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/schema"
)

// LayoutGroup renders a vertical widget group in HTML.
func LayoutGroup(library form.Library) {

	library.Register("layout-group", func(form form.Form, schema schema.Schema, value interface{}, builder *strings.Builder) error {

		var result error

		builder.WriteString(`<div class="layout-group">`)

		builder.WriteString(`<div class="label">` + html.EscapeString(form.Label) + `</div>`)
		builder.WriteString(`<div class="elements>`)

		for index, child := range form.Children {
			builder.WriteString(`<div class="element">`)

			if err := child.Write(library, schema, value, builder); err != nil {
				result = derp.Wrap(err, "form.widget.LayoutGroup", "Error rendering child", index, child)
			}
			builder.WriteString(`</div>`)
		}

		builder.WriteString(`</div>`)
		builder.WriteString(`</div>`)

		return result
	})
}
