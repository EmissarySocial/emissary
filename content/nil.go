package content

import "github.com/benpate/html"

type Nil struct{}

func (widget Nil) View(b *html.Builder, c Content, id int) {
	// Nothing to see here
}

func (widget Nil) Edit(b *html.Builder, c Content, id int, endpoint string) {
	// Nothing to see here
}
