package camper

// CreateIntent defines the structure for a "Create" intent
type CreateIntent struct {
	ObjectType string `query:"type"       form:"type"`
	Name       string `query:"name"       form:"name"`
	Summary    string `query:"summary"    form:"summary"`
	Content    string `query:"content"    form:"content"`
	InReplyTo  string `query:"inReplyTo"  form:"inReplyTo"`
	OnSuccess  string `query:"on-success" form:"on-success"`
	OnCancel   string `query:"on-cancel"  form:"on-cancel"`
}

// DislikeIntent defines the structure for a "Dislike" intent
type DislikeIntent struct {
	Object    string `query:"object"     form:"object"`
	OnSuccess string `query:"on-success" form:"on-success"`
	OnCancel  string `query:"on-cancel"  form:"on-cancel"`
}

// LikeIntent defines the structure for a "Like" intent
type LikeIntent struct {
	Object    string `query:"object" form:"object"`
	OnSuccess string `query:"on-success" form:"on-success"`
	OnCancel  string `query:"on-cancel"  form:"on-cancel"`
}
