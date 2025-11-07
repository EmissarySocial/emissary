package consumer

import (
	"github.com/benpate/derp"
	"github.com/benpate/turbine/queue"
)

// PreProcessor defines global rules for all tasks in the system
func PreProcessor(task *queue.Task) error {

	const location = "consumer.PrePreprocessor"

	switch task.Name {

	///////////////////////////////////////////////////
	// Tasks with priority <= 32 are executed
	// immediately IF the queue is not already busy

	// (8) User-Facing Tasks that affect UX
	case "CreateWebSubFollower":
		task.Priority = 8

	// (16) Realtime User Notifications
	case "SendActivityPubMessage":
		task.Priority = 16

	case "SendSearchResult":
		task.Priority = 16

	case "SendSearchResult-SearchQuery":
		task.Priority = 16

	///////////////////////////////////////////////////
	// Tasks below this line are ALWAYS written to the
	// database, and are NOT executed immediately

	// (64) User Facine but Low Priority Tasks
	case "CrawlActivityStreams":
		task.Priority = 64

	case "Geocode":
		task.Priority = 64

	// (256) Background Notifications
	case "MakeStreamArchive":
		task.Priority = 256

	case "ReceiveWebMention":
		task.Priority = 256

	case "SendWebMention":
		task.Priority = 256

	case "SendWebSubMessage":
		task.Priority = 256

	case "syndication.create", "syndication.update", "syndication.delete":
		task.Priority = 256

	// (512) System Tasks that should happen mostly on time
	case "DeleteStream":
		task.Priority = 512

	case "LoadActivityStream":
		task.Priority = 512

	case "PollFollowing":
		task.Priority = 512

	case "ReindexActivityStream":
		task.Priority = 512

	case "Scheduler":
		task.Priority = 512

	case "ScheduleStartup":
		task.Priority = 512

	case "ScheduleDaily":
		task.Priority = 512

	case "ScheduleHourly":
		task.Priority = 512

	case "Shuffle":
		task.Priority = 512

	// 1024: Daily/Hourly Tasks that can happen whenever it's convenient
	case "DeleteEmptySearchQuery":
		task.Priority = 1024

	case "PurgeActivityStreamCache":
		task.Priority = 1024

	case "PurgeErrors":
		task.Priority = 1024

	case "PurgeDomeLog":
		task.Priority = 1024

	case "RecycleDomain":
		task.Priority = 1024

	// Blocked Tasks
	case "CountRelatedDocuments":
		return derp.NotImplemented(location, "CountRelatedDocuments task has been disabled")

	case "ProcessMedia":
		return derp.NotImplemented(location, "ProcessMedia task has not been implemented")

	}

	return nil
}
