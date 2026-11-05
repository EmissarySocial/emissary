package consumer

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
)

// stagedUploadRetention is how long a file staged for a form that has not been saved is kept.
// A form left open overnight keeps its files; one that was abandoned does not keep them forever.
const stagedUploadRetention = 24 * time.Hour

// PurgeStagedUploads deletes the files that edit-attachments staged for forms that were
// never saved, once they are older than stagedUploadRetention
func PurgeStagedUploads(factory *service.Factory, session data.Session, _ mapof.Any) queue.Result {

	const location = "consumer.PurgeStagedUploads"

	log.Trace().Msg("Task: PurgeStagedUploads")

	count, err := factory.Attachment().PurgeStaged(session, model.AttachmentCategoryStaging, time.Now().Add(-stagedUploadRetention))

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Purging staged uploads"))
	}

	log.Debug().Int("count", count).Msg("Purged staged uploads")

	return queue.Success()
}
