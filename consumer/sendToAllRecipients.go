package consumer

import (
	"github.com/benpate/hannibal/sender"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// SendToAllRecipients delivers an ActivityPub message to every recipient named in the task arguments
func SendToAllRecipients(sender sender.Sender, args mapof.Any) queue.Result {
	return sender.SendToAllRecipients(args)
}
