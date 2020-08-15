package vocabulary

import (
	"strings"

	"github.com/benpate/form"
	"github.com/benpate/schema"
)

// Textarea registers a <textarea> input widget into the library
func Textarea(library form.Library) {

	library.Register("textarea", func(f form.Form, s schema.Schema, v interface{}, builder *strings.Builder) error {

		// find the path and schema to use
		schemaObject, valueString := locateSchema(f.Path, s, v)

		// Start building a new tag
		tag := TagBuilder("textarea", builder)

		// Always dd ID attribute (if values exist)
		tag.Attr("id", f.ID)
		tag.Attr("name", f.Path)
		tag.Attr("class", "uk-textarea")

		// Add attributes that depend on what KIND of input we have.
		if schemaString, ok := schemaObject.(schema.String); ok {

			if schemaString.MinLength.IsPresent() {
				tag.Attr("minlength", schemaString.MinLength.Int())
			}

			if schemaString.MaxLength.IsPresent() {
				tag.Attr("maxlength", schemaString.MaxLength.Int())
			}

			if schemaString.Pattern != "" {
				tag.Attr("pattern", schemaString.Pattern)
			}

			if schemaString.Required {
				tag.Attr("required", true)
			}
		}

		if f.CSSClass != "" {
			tag.Attr("class", f.CSSClass)
		}

		if f.Description != "" {
			tag.Attr("hint", f.Description)
		}

		tag.InnerHTML(valueString)

		tag.EndTag()

		return nil
	})
}
