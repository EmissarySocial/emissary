package vocabulary

import (
	"html"
	"strings"

	"github.com/benpate/convert"
)

// Tag represents a tag that is being written into the provided strings.Builder
type Tag struct {
	name    string
	builder *strings.Builder
	closed  bool
	endTag  bool
}

// TagBuilder returns a Tag that can be written into the provided srings.Builder
func TagBuilder(name string, builder *strings.Builder) *Tag {

	result := Tag{
		name:    name,
		builder: builder,
		closed:  false,
	}

	// grow length of "<" + name + ">"
	result.builder.Grow(len(name) + 2)
	result.builder.WriteRune('<')
	result.builder.WriteString(name)
	return &result
}

// Attr writes the attribute into the string builder.  It converts
// the value (second parameter) into a string, and then uses html.EscapeString
// to escape the attribute value.  Attribute names ARE NOT escaped.
func (tag *Tag) Attr(name string, value interface{}) *Tag {

	// We can only write more attributes if the tag has not yet been closed
	if tag.closed == false {

		// this *should* already be a string, but just in case
		if valueString, _ := convert.StringOk(value, ""); valueString != "" {

			// escape the value
			valueString = html.EscapeString(valueString)

			// length of: space + name + quote + escaped value + quote
			tag.builder.Grow(len(name) + len(valueString) + 4)

			// write values to the builder
			tag.builder.WriteRune(' ')
			tag.builder.WriteString(name)
			tag.builder.WriteString(`="`)
			tag.builder.WriteString(valueString)
			tag.builder.WriteRune('"')
		}
	}

	return tag
}

// Close writes the final ">" of the beginning tag to the strings.Builder
// It uses an internal variable to prevent duplicate calls
func (tag *Tag) Close() *Tag {

	if tag.closed == false {
		tag.closed = true
		tag.builder.WriteRune('>')
	}

	return tag
}

// InnerHTML does three things:
// 1) closes the beginning tag (if needed)
// 2) appends innerHTML (if provided)
// 3) writes an ending tag to the builder (ie. </tag> )
func (tag *Tag) InnerHTML(innerHTML string) {

	// If an endTag has already been written, then we cannot write any more.
	if tag.endTag {
		return
	}

	growSize := len(tag.name) + len(innerHTML) + 3

	// If the tag is not already closed, then close it now.
	if tag.closed == false {
		tag.builder.Grow(growSize + 1)
		tag.builder.WriteRune('>')
	} else {
		tag.builder.Grow(growSize)
	}

	// Write innerHTML (if present)
	if innerHTML != "" {
		tag.builder.WriteString(innerHTML)
	}

	// Write the remaining end tag
	tag.builder.WriteString("</")
	tag.builder.WriteString(tag.name)
	tag.builder.WriteRune('>')

	// Mark this tag as closed and ended.
	tag.closed = true
	tag.endTag = true
}

// EndTag does two things:
// 1) closes the beginning tag (if needed)
// 2) writes an ending tag to the builder (ie. </tag> )
func (tag *Tag) EndTag() {

	// This is semantically the same thing.  Just syntactic sugar to make source code read better.
	tag.InnerHTML("")
}
