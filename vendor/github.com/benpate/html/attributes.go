package html

// Class adds a "class" attribute to the Element
func (element *Element) Class(value string) *Element {
	return element.Attr("class", value)
}

// For adds a "for" attribute to the Element
func (element *Element) For(value string) *Element {
	return element.Attr("for", value)
}

// Data adds a "data-" attribute to the Element
func (element *Element) Data(name string, value string) *Element {
	return element.Attr("data-"+name, value)
}

// ID adds an "id" attribute to the Element
func (element *Element) ID(value string) *Element {
	return element.Attr("id", value)
}

// Label adds a "label" attribute to the Element
func (element *Element) Label(value string) *Element {
	return element.Attr("label", value)
}

// List adds a "list" attribute to the Element
func (element *Element) List(value string) *Element {
	return element.Attr("list", value)
}

// Name adds a "name" attribute to the Element
func (element *Element) Name(value string) *Element {
	return element.Attr("name", value)
}

// Script adds a "data-script" attribute to the Element (hyperscript)
func (element *Element) Script(value string) *Element {
	return element.Attr("_", value)
}

// Type adds a "type" attribute to the Element
func (element *Element) Type(value string) *Element {
	return element.Attr("type", value)
}

// Value adds a "value" attribute to the Element
func (element *Element) Value(value string) *Element {
	return element.Attr("value", value)
}
