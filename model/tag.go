package model

import (
	"strings"

	"github.com/benpate/rosetta/mapof"
)

type Tag struct {
	Type string `json:"type"` // Type of Tag (e.g. "Hashtag", "Mention")
	Name string `json:"name"` // Value to display (e.g. "#hashtag", "@mention")
	Href string `json:"href"` // URL to link to (e.g. "/hashtag/hashtag", "/user/username")
}

func NewTag() Tag {
	return Tag{}
}

func (tag Tag) JSONLD() mapof.Any {
	return TagAsJSONLD(tag)
}

func TagAsJSONLD(tag Tag) mapof.Any {
	return mapof.Any{
		"type": tag.Type,
		"name": tag.Name,
		"href": tag.Href,
	}
}

func TagAsNameOnly(tag Tag) string {
	result := tag.Name

	result = strings.TrimPrefix(result, "#")
	result = strings.TrimPrefix(result, "@")
	return result
}
