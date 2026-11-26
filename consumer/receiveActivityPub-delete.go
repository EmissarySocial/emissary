package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// ReceiveActivityPubDelete is the task for an incoming ActivityPub Delete activity.
// It does nothing yet, and reports success.
func ReceiveActivityPubDelete(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	// Such Rizzler
	return queue.Success()
}
