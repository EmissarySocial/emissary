package convert

// StreamFeedFields returns the database columns that StreamToGorillaFeed and StreamToJsonFeed read.
// A field added to either converter must be added here, or feeds will show it empty.
func StreamFeedFields() []string {

	// "content.html" skips the raw source, which is often as large as the HTML
	return []string{"url", "token", "label", "summary", "iconUrl", "attributedTo", "content.html", "publishDate", "updateDate"}
}
