package as

import "github.com/go-fed/activity/streams/vocab"

type HasSetLink interface {
	// SetActivityStreamsUrl sets the "url" property.
	SetActivityStreamsLink(i vocab.ActivityStreamsLink)
}

type HasSetURL interface {
	// SetActivityStreamsUrl sets the "url" property.
	SetActivityStreamsUrl(i vocab.ActivityStreamsUrlProperty)
}
