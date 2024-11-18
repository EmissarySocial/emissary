package consumer

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

func ProcessMedia(factory *domain.Factory, args mapof.Any) queue.Result {

	return queue.Success()
}
