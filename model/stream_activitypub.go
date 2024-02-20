package model

func (stream *Stream) ActivityPubID() string {
	return stream.URL
}

func (stream *Stream) ActivityPubOutboxURL() string {
	return stream.URL + "/pub/outbox"
}

func (stream *Stream) ActivityPubInboxURL() string {
	return stream.URL + "/pub/inbox"
}

func (stream *Stream) ActivityPubFollowersURL() string {
	return stream.URL + "/pub/followers"
}
