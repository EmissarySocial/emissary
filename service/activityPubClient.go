package service

import (
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/mapof"
)

type ActivityPubClient struct{}

func NewActivityPubClient() *ActivityPubClient {
	return &ActivityPubClient{}
}

func (client *ActivityPubClient) Load(uri string) (streams.Document, error) {

	// TODO: MEDIUM: Add memory caching / Database caching here.

	result := mapof.NewAny()

	transaction := remote.
		Get(uri).
		Header("Accept", "application/activity+json").
		Response(&result, nil)

	if err := transaction.Send(); err != nil {
		return streams.NilDocument(), derp.Wrap(err, "service.ActivityPubClient.Load", "Error sending request")
	}

	document := streams.NewDocument(result, streams.WithClient(client))
	return document, nil
}
