package model

import "github.com/benpate/rosetta/mapof"

type JSONLD mapof.Any

func (j JSONLD) GetJSONLD() mapof.Any {
	return mapof.Any(j)
}

func (j JSONLD) Created() int64 {
	return mapof.Any(j).GetInt64("published")
}
