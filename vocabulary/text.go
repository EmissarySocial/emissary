package vocabulary

import (
	"strings"

	"github.com/benpate/convert"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"github.com/benpate/schema"
)

// Text registers a text <input> widget into the library
func Text(library form.Library) {

	library.Register("text", func(f form.Form, s *schema.Schema, v interface{}, b *html.Builder) error {

		var listID string

		// find the path and schema to use
		schemaElement, value := locateSchema(f.Path, s, v)

		valueString := convert.String(value)

		// Start building a new tag
		tag := b.Input("text", f.Path).
			ID(f.ID).
			Value(valueString)

		// Enumeration Options
		options := library.Options(f, schemaElement)

		if len(options) > 0 {
			if f.ID != "" {
				listID = "datalist_" + f.ID
			} else {
				listID = "datalist_" + strings.ReplaceAll(f.Path, "/", "_")
			}
			tag.Attr("list", listID)
		}

		// Add attributes that depend on what KIND of input we have.
		switch s := schemaElement.(type) {

		case schema.Integer:
			tag.Type("number").Attr("step", "1")

			if s.Minimum.IsPresent() {
				tag.Attr("min", s.Minimum.String())
			}

			if s.Maximum.IsPresent() {
				tag.Attr("max", s.Maximum.String())
			}

			if s.Required {
				tag.Attr("required", "true")
			}

		case schema.Number:

			tag.Type("number")

			if s.Minimum.IsPresent() {
				tag.Attr("min", s.Minimum.String())
			}

			if s.Maximum.IsPresent() {
				tag.Attr("max", s.Maximum.String())
			}

			if s.Required {
				tag.Attr("required", "true")
			}

		case schema.String:

			switch s.Format {
			case "email":
				tag.Type("email")
			case "tel":
				tag.Type("tel")
			case "url":
				tag.Type("url")
			default:
				tag.Type("text")
			}

			if s.MinLength.IsPresent() {
				tag.Attr("minlength", s.MinLength.String())
			}

			if s.MaxLength.IsPresent() {
				tag.Attr("maxlength", s.MaxLength.String())
			}

			if s.Pattern != "" {
				tag.Attr("pattern", s.Pattern)
			}

			if s.Required {
				tag.Attr("required", "true")
			}

		default:
			tag.Type("text")
		}

		if f.CSSClass != "" {
			tag.Attr("class", f.CSSClass)
		}

		if f.Description != "" {
			tag.Attr("hint", f.Description)
		}

		tag.Close()

		if len(options) > 0 {
			b.Container("datalist").ID(listID)
			for _, option := range options {
				b.Empty("option").Value(option.Value).Close()
			}
			b.Close()
		}

		b.CloseAll()
		return nil
	})
}

/*


   <input type="button">
   <input type="checkbox">
   <input type="color">
   <input type="date">
   <input type="datetime-local">
   <input type="email">
   <input type="file">
   <input type="hidden">
   <input type="image">
   <input type="month">
   <input type="number">
   <input type="password">
   <input type="radio">
   <input type="range">
   <input type="reset">
   <input type="search">
   <input type="submit">
   <input type="tel">
   <input type="text">
   <input type="time">
   <input type="url">
   <input type="week">
*/
