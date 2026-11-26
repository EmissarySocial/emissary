package convert

// StreamFeedFields returns the database columns that StreamToGorillaFeed and StreamToJsonFeed read.
// A field added to either converter must be added here, or feeds will show it empty.
func StreamFeedFields() []string {

	// "content.html" skips the raw source, which is often as large as the HTML
	// "data.enclosureUrl" and "data.enclosureType" are a podcast episode's audio (enclosure.go)
	// "createDate" dates a page that never entered the outbox, which has no "publishDate"
	return []string{"url", "token", "label", "summary", "iconUrl", "attributedTo", "content.html", "publishDate", "createDate", "updateDate", "data.enclosureUrl", "data.enclosureType"}
}
