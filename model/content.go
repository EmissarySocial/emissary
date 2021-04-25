package model

type Content struct {
	Type string `json:"type" bson:"type"` // Identifies the kind of editor to use.
	HTML string `json:"html" bson:"html"` // HTML content to display in the stream.
}
