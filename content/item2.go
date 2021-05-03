package content

import "github.com/benpate/html"

type Widget2 interface {
	Default() Item
	View(*Library, *html.Builder, Content, int)
	Edit(*Library, *html.Builder, Content, int)
	Transaction(*Content, Transaction)
}
