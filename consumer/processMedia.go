package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

func ProcessMedia(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	return queue.Success()
}
