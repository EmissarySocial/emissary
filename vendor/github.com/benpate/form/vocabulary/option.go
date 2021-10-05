package vocabulary

import (
	"github.com/benpate/convert"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"github.com/benpate/schema"
)

// Option registers a text <input> widget into the library
func Option(library form.Library) {

	library.Register("option", func(f form.Form, s *schema.Schema, v interface{}, b *html.Builder) error {

		// find the path and schema to use
		schemaElement, value := locateSchema(f.Path, s, v)
		valueString := convert.String(value)

		format := f.Options["format"]

		if format == "" {

			if schemaElement.Type() == schema.TypeArray {
				format = "checkbox"
			} else {
				format = "radio"
			}
		}

		// Start building a new tag

		b.Label(f.ID)

		b.Input(format, f.Path).
			ID(f.ID).
			Value(valueString).
			Class(f.CSSClass).
			InnerHTML(f.Label)

		b.CloseAll()

		return nil
	})
}
