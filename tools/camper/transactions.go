package camper

// CreateIntent defines the structure for a "Create" intent
type CreateIntent struct {
	ObjectType string `url:"type"       form:"type"`
	Name       string `url:"name"       form:"name"`
	Summary    string `url:"summary"    form:"summary"`
	Content    string `url:"content"    form:"content"`
	InReplyTo  string `url:"inReplyTo"  form:"inReplyTo"`
	OnSuccess  string `url:"on-success" form:"on-success"`
	OnCancel   string `url:"on-cancel"  form:"on-cancel"`
}

// DislikeIntent defines the structure for a "Dislike" intent
type DislikeIntent struct {
	Object    string `url:"object" form:"object"`
	OnSuccess string `url:"on-success" form:"on-success"`
	OnCancel  string `url:"on-cancel"  form:"on-cancel"`
}

// LikeIntent defines the structure for a "Like" intent
type LikeIntent struct {
	Object    string `url:"object" form:"object"`
	OnSuccess string `url:"on-success" form:"on-success"`
	OnCancel  string `url:"on-cancel"  form:"on-cancel"`
}
