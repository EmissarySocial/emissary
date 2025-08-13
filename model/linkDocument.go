package model

type DocumentLink struct {
	ID           string     // ID of the record that is being linked (if different from its URL)
	URL          string     // URL of the original document
	Name         string     // Label/Title of the document
	Icon         string     // URL of the icon image for this document
	Summary      string     // Brief summary of the document
	Content      string     // Full content of the document
	AttributedTo PersonLink // Person that this document is attributed to
	Published    int64      // Timestamp of when the document was published
}

func NewDocumentLink() DocumentLink {
	return DocumentLink{
		AttributedTo: NewPersonLink(),
	}
}
