package model

type Content struct {
	HTML    string `json:"html" form:"html"    bson:"html"`    // HTML content to display in the stream.
	Hash    string `json:"hash" form:"hash"    bson:"hash"`    // Edit counter (for optimistic locking and JWT authentication)
	Content string `json:"-"    form:"content" bson:"content"` // Key used to located this content. (In-Memory Only)
	Stream  string `json:"-"    form:"-"       bson:"-"`       // Token used to locate the stream. (In-Memory Only)
	Editor  string `json:"-"    form:"-"       bson:"-"`       // Identifies the kind of editor to use.
}
