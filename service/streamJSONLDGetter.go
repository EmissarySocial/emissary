package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/rosetta/mapof"
)

// StreamJSONLDGetter wraps the Stream service and a model.Stream to provide a JSONLDGetter interface
type StreamJSONLDGetter struct {
	session       data.Session
	streamService *Stream
	stream        *model.Stream
}

// NewStreamJSONLDGetter returns a fully initialized StreamJSONLDGetter
func NewStreamJSONLDGetter(session data.Session, streamService *Stream, stream *model.Stream) StreamJSONLDGetter {
	return StreamJSONLDGetter{
		session:       session,
		streamService: streamService,
		stream:        stream,
	}
}

// GetJSONLD returns a JSON-LD representation of the wrapped Stream
func (getter StreamJSONLDGetter) GetJSONLD() mapof.Any {
	return getter.streamService.JSONLD(getter.session, getter.stream)
}

// Created returns the creation date of the wrapped Stream
func (getter StreamJSONLDGetter) Created() int64 {
	return getter.stream.CreateDate
}
