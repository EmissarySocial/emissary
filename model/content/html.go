package content

// HTML represents raw HTML in the CMS
type HTML string

// HTML implements the Content interface
func (html *HTML) HTML() string {
	return string(*html)
}

// WebComponents accumulates all of the scripts that are required to correctly render the HTML for this content object
func (html *HTML) WebComponents(accumulator map[string]bool) {
	return
}
