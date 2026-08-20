package consumer

import (
	"github.com/benpate/hannibal/sender"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// SendToSingleRecipient delivers an ActivityPub message to the one recipient named in the task arguments
func SendToSingleRecipient(sender sender.Sender, args mapof.Any) queue.Result {
	return sender.SendToSingleRecipient(args)
}
