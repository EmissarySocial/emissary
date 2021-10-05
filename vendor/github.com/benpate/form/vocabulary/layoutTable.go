package vocabulary

// THIS WILL TAKE SOME EXTRA WORK BECAUSE WE NEED TO FIGURE OUT HOW TO PASS THE ARRAY INDEX INTO EACH ROW.

/*

func LayoutTable(library form.Library) {

	library.Register("layout-table", func(form form.Form, schema schema.Schema, value interface{}, builder *strings.Builder) error {

		var result error

		builder.WriteString(`<table class="layout-table"><tr class="head">`)

		for _, child := range form.Children {
			builder.WriteString("<td>" + html.EscapeString(child.Label) + "</td>")
		}
		builder.WriteString(`</tr>`)

		for index, child := range form.Children {
			builder.WriteString(`<div class="element">`)

			if err := child.Write(library, schema, value, builder); err != nil {
				result = derp.Wrap(err, "form.widget.LayoutGroup", "Error rendering child", index, element)
			}
			builder.WriteString(`</div>`)
		}

		builder.WriteString(`</table>`)

		return result
	})
}
*/
