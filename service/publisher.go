package service

import (
	"github.com/benpate/activitystream/writer"
	"github.com/benpate/data"
)

// Publisher service knows how to publish ActivityPub events based on subscriptions that are registered in the database
type Publisher struct {
	factory Factory
	session data.Session
}

// Publish sends notifications to external services when an event occurs.
func (publisher Publisher) Publish(writer.Object) error {

	/* TODO:  This should be asynchrous.

	get subscriptions

	if subscriptions.length {

		for index, subscription := range subscriptions {

		}
	}
	*/

	return nil
}
