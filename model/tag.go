package model

type Tag struct {
	Type string `json:"type"` // Type of Tag (e.g. "Hashtag", "Mention")
	Name string `json:"name"` // Value to display (e.g. "#hashtag", "@mention")
	Href string `json:"href"` // URL to link to (e.g. "/hashtag/hashtag", "/user/username")
}

func NewTag() Tag {
	return Tag{}
}
