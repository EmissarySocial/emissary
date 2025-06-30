package model

import (
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
)

type JSONLD mapof.Any

func (j JSONLD) GetJSONLD() mapof.Any {
	return mapof.Any(j)
}

func (j JSONLD) ActivityPubURL() string {
	return mapof.Any(j).GetString(vocab.PropertyID)
}

func (j JSONLD) Created() int64 {
	return mapof.Any(j).GetInt64(vocab.PropertyPublished)
}
