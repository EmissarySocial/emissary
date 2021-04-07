package vocabulary

import (
	"github.com/benpate/compare"
	"github.com/benpate/convert"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"github.com/benpate/schema"
)

// Select registers a text <input> widget into the library
func Select(library form.Library) {

	library.Register("select", func(f form.Form, s *schema.Schema, v interface{}, b *html.Builder) error {

		var selectMany bool

		// find the path and schema to use
		schemaElement, value := locateSchema(f.Path, s, v)

		if element, ok := schemaElement.(schema.Array); ok {
			schemaElement = element.Items
			selectMany = true
		}

		// Get all options for this element...
		options := library.Options(f, schemaElement)

		// SelectMany
		if selectMany {

			valueSlice := convert.SliceOfString(value)

			for _, option := range options {
				label := b.Label(f.ID)

				input := b.Input().ID(f.ID).Name(f.Path).Value(option.Value).Attr("type", "checkbox")

				if compare.Contains(valueSlice, option.Value) {
					input.Attr("checked", "true")
				}

				input.Close()
				label.InnerHTML(option.Label)
				label.Close()
			}

			b.CloseAll()
			return nil
		}

		// SelectOne
		valueString := convert.String(value)

		if f.Options["format"] == "radio" {

			for _, option := range options {
				label := b.Label(f.ID)

				input := b.Input().
					ID(f.ID).
					Name(f.Path).
					Value(option.Value).
					Type("radio")

				if valueString == option.Value {
					input.Attr("checked", "true")
				}

				input.Close()
				label.InnerHTML(option.Label)
				label.Close()
			}

		} else {

			// Fall through to select box

			dropdown := b.Container("select").ID(f.ID).Name(f.Path).Class(f.CSSClass)

			for _, option := range options {
				opt := b.Container("option").Value(option.Value)
				if option.Value == valueString {
					opt.Attr("selected", "true")
				}
				opt.InnerHTML(option.Label).Close()
			}
			dropdown.Close()
		}

		b.CloseAll()
		return nil
	})
}
