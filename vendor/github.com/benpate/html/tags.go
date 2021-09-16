package html

func (b *Builder) A(href string) *Element {
	return b.Container("a").Attr("href", href)
}

func (b *Builder) BR() *Element {
	return b.Empty("br")
}

func (b *Builder) Button() *Element {
	return b.Container("button")
}

func (b *Builder) Datalist(id string) *Element {
	return b.Container("datalist").ID(id).EndBracket()
}

func (b *Builder) Div() *Element {
	return b.Container("div")
}

func (b *Builder) Form(method string, action string) *Element {
	return b.Container("form").Attr("method", method).Attr("action", action)
}

func (b *Builder) Input(t string, name string) *Element {
	return b.Empty("input").Type(t).Name(name)
}

func (b *Builder) Label(forID string) *Element {
	return b.Container("label").For(forID)
}

func (b *Builder) OptGroup(label string) *Element {
	return b.Container("optgroup").Label(label).EndBracket()
}

func (b *Builder) Option(name string, value string) *Element {
	return b.Container("option").Name(name).InnerHTML(value).Close()
}

func (b *Builder) Script() *Element {
	return b.Container("script")
}

func (b *Builder) Select(name string) *Element {
	return b.Container("select").Name(name)
}

func (b *Builder) Span() *Element {
	return b.Container("span")
}

func (b *Builder) Textarea(name string) *Element {
	return b.Container("textarea").Name(name)
}
