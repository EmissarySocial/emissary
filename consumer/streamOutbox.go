package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// StreamOutbox brings a Stream's followers up to date after a save.  It sends nothing yet:
// the publish steps still send, and this task takes over from them.
func StreamOutbox(_ *service.Factory, _ data.Session, _ mapof.Any) queue.Result {

	// Nothing to see here.  Move along.
	return queue.Success()
}
