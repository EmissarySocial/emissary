package model

import (
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
)

// JSONLD returns this  as a JSON-LD map
type JSONLD mapof.Any

// GetJSONLD returns this JSONLD as a JSON-LD document
func (j JSONLD) GetJSONLD() mapof.Any {
	return mapof.Any(j)
}

// ActivityPubURL returns the ActivityPub URL of this JSONLD
func (j JSONLD) ActivityPubURL() string {
	return mapof.Any(j).GetString(vocab.PropertyID)
}

// Created returns the creation date of this JSONLD, as a Unix timestamp
func (j JSONLD) Created() int64 {
	return mapof.Any(j).GetInt64(vocab.PropertyPublished)
}
