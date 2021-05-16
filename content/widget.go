package content

import "github.com/benpate/html"

type Widget interface {
	View(*html.Builder, Content, int)
	Edit(*html.Builder, Content, int, string)
}

type DefaultChildrener interface {
	DefaultChildren() []Item
}
