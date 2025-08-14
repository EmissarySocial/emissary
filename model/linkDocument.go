package model

type DocumentLink struct {
	ID           string     // ID of the record that is being linked (if different from its URL)
	InReplyTo    string     // ID of the document this one is replying to (if applicable)
	Token        string     // Other token to use when identifying this document (like a hashed-id or URL slug)
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

func (document DocumentLink) TreeID() string {
	return document.ID
}

func (document DocumentLink) TreeParent() string {
	return document.InReplyTo
}
