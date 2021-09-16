package html

import (
	"html"
)

// Element represents a element that is being written into the provided strings.Builder
type Element struct {
	builder    *Builder
	parent     *Element
	name       string
	container  bool
	endBracket bool
	closed     bool
}

// Start writes the initial tag name and opening bracket for this tag.
func (element *Element) Start() *Element {

	var growSize int

	if element.container {
		growSize = (2*len(element.name) + 2) + 1 // < + name + > + </ + name + >
	} else {
		growSize = len(element.name) + 2 // < + name + >
	}

	// Grow the buffer and write the new tag name
	element.builder.Grow(growSize)
	element.builder.WriteRune('<')
	element.builder.WriteString(element.name)

	return element
}

// Attr writes the attribute into the string builder.  It converts
// the value (second parameter) into a string, and then uses html.EscapeString
// to escape the attribute value.  Attribute names ARE NOT escaped.
func (element *Element) Attr(name string, value string) *Element {

	// If the element already has an end bracket, then we can't add any more attributes.
	if element.endBracket {
		return element
	}

	// this *should* already be a string, but just in case
	if value != "" {

		// escape the value
		value = html.EscapeString(value)

		// length of: space + name + quote + escaped value + quote
		element.builder.Grow(len(name) + len(value) + 4)

		// write values to the builder
		element.builder.WriteRune(' ')
		element.builder.WriteString(name)
		element.builder.WriteString(`="`)
		element.builder.WriteString(value)
		element.builder.WriteRune('"')
	}

	return element
}

// EndBracket writes the final ">" of the beginning element to the strings.Builder
// It uses an internal variable to prevent duplicate calls
func (element *Element) EndBracket() *Element {

	// If we already have an end bracket, then skip
	if element.endBracket {
		return element
	}

	// If this element is not a container, then this closes it permanently
	if !element.container {
		element.closed = true
	}

	element.endBracket = true
	element.builder.WriteRune('>')
	return element
}

// InnerHTML does three things:
// 1) closes the beginning element (if needed)
// 2) appends innerHTML (if provided)
// 3) writes an ending element to the builder (ie. </element> )
func (element *Element) InnerHTML(innerHTML string) *Element {

	// If the element has already been closed, then we cannot add anything more.
	if element.closed {
		return element
	}

	// Only need to write additional content if innerHTML is not empty
	if innerHTML != "" {

		if !element.endBracket {
			element.builder.WriteRune('>')
			element.endBracket = true
		}

		// Write innerHTML (if present)
		element.builder.Grow(len(innerHTML))
		element.builder.WriteString(innerHTML)
	}

	// Write the remaining end element
	return element.Close()
}

// Close writes the necessary closing tag for this element and marks it closed
func (element *Element) Close() *Element {

	// If it's already been closed, then nothing else is required.
	if element.closed {
		return element
	}

	// Mark this element as closed.
	element.EndBracket()

	// If this is a CONTAINER element, then add the ending tag too.
	if element.container {
		// correct buffer size is set when we create the tag...
		element.builder.WriteString("</")
		element.builder.WriteString(element.name)
		element.builder.WriteRune('>')
	}

	// Mark the element closed
	element.closed = true

	// Update the builder's stack.
	element.builder.last = element.parent

	return element
}
