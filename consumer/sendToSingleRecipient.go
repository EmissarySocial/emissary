package consumer

import (
	"github.com/benpate/hannibal/sender"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

func SendToSingleRecipient(sender sender.Sender, args mapof.Any) queue.Result {
	return sender.SendToSingleRecipient(args)
}
