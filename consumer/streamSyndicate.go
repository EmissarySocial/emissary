package consumer

import (
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/mapof"
)

// StreamSyndicate sends HTTP messages to syndication targets
func StreamSyndicate(args mapof.Any) error {

	// Find the endpoint Href for the selected syndication target
	endpoint := args.GetString("endpoint")
	message := args.GetMap("message")

	// Create and send the message to the remote endpoint
	txn := remote.Post(endpoint).JSON(message)

	if err := txn.Send(); err != nil {
		return derp.Wrap(err, "consumer.StreamSyndicate", "Error sending syndication message")
	}

	// Success!
	return nil
}
