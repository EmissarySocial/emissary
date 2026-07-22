package replace

import (
	stdhtml "html"
	"net/url"
)

// Linkify wraps each occurrence of "#"+tag in the html with an anchor pointing at baseURL.  The tag
// is URL-escaped in the href and HTML-escaped in the visible label, so a database-sourced tag name
// cannot break out of the attribute or inject markup.  Because it uses Content, tags already inside
// an <a> element are left untouched, making repeated calls idempotent.
func Linkify(html string, baseURL string, tags []string) string {

	for _, tag := range tags {

		if tag == "" {
			continue
		}

		href := baseURL + "%23" + url.QueryEscape(tag)
		label := stdhtml.EscapeString("#" + tag)

		html = Content(html, "#"+tag, `<a href="`+href+`" target="_blank">`+label+`</a>`)
	}

	return html
}
