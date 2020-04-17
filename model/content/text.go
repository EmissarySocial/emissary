package content

import "github.com/benpate/html"

// Text represents simple text content that can be rendered into HTML
type Text string

// HTML implements the HTMLer interface
func (text *Text) HTML() string {
	return html.FromText(string(*text))
}

// WebComponents accumulates all of the scripts that are required to correctly render the HTML for this content object
func (text *Text) WebComponents(accumulator map[string]bool) {
	return
}
