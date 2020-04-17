// Package content defines all of the kinds of content that this server supports.  Content types are stored
// in the database, and can be rendered into HTML directly on the server.
package content

// Content is the basic interface that every content type must support
type Content interface {
	HTML() string
	WebComponents(map[string]bool)
}

// Parse takes an arbitrary data set and attempts to parse it into a single Content value
func Parse(_ interface{}) (Content, bool) {

	result := HTML("<example></example>")

	return &result, false
}
