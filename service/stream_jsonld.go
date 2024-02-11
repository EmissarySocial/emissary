package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
)

type StreamJSONLDGetter struct {
	streamService *Stream
	stream        *model.Stream
}

func NewStreamJSONLDGetter(streamService *Stream, stream *model.Stream) *StreamJSONLDGetter {
	return &StreamJSONLDGetter{
		streamService: streamService,
		stream:        stream,
	}
}

func (getter StreamJSONLDGetter) GetJSONLD() mapof.Any {
	return getter.streamService.JSONLD(getter.stream)
}

func (getter StreamJSONLDGetter) Created() int64 {
	return getter.stream.CreateDate
}
