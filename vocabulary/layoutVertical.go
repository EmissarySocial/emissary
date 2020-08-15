package vocabulary

import (
	"html"
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/schema"
)

// LayoutVertical defines a standard top to bottom layout, including labels above every child item.
func LayoutVertical(library form.Library) {

	library.Register("layout-vertical", func(form form.Form, schema schema.Schema, value interface{}, builder *strings.Builder) error {

		var result error

		builder.WriteString(`<div class="uk-form-stacked">`)

		if len(form.Label) > 0 {
			builder.WriteString(`<div class="label">`)
			builder.WriteString(html.EscapeString(form.Label))
			builder.WriteString(`</div>`)
		}

		for index, child := range form.Children {

			builder.WriteString(`<div class="uk-margin">`)

			TagBuilder("label", builder).Attr("for", child.ID).Attr("class", "uk-form-label").InnerHTML(child.Label)
			TagBuilder("div", builder).Attr("class", "uk-form-controls").Close()

			if err := child.Write(library, schema, value, builder); err != nil {
				result = derp.Wrap(err, "form.widget.LayoutVertical", "Error rendering child", index, form)
			}

			builder.WriteString(`</div>`)
			builder.WriteString(`</div>`)
		}

		builder.WriteString(`</div>`)

		return result
	})
}
