package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// ReceiveActivityPubDelete processes an incoming ActivityPub Delete activity
// by verifying the the original object no longer exists, then removing it from the ActivityPub cache.
func ReceiveActivityPubDelete(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	// Such Rizzler
	return queue.Success()
}
