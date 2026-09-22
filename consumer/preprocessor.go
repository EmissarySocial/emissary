package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/turbine/queue"
)

// PreProcessor defines global rules for all tasks in the system
func PreProcessor(task *queue.Task) error {

	// If the priority has already been set, then leave it alone
	if task.Priority != -1 {
		return nil
	}

	switch task.Name {

	///////////////////////////////////////////////////
	// Tasks with priority <= 32 are executed
	// immediately IF the queue is not already busy

	// (8) User-Facing Tasks that affect UX
	case "ConnectPushService":
		task.Priority = 8

	// (16) Realtime User Notifications
	case "ImportStartup":
		task.Priority = 16

	// Outbox-Publish is the follower fan-out (F2). It is cheap (DB reads + enqueue, no HTTP),
	// so it runs promptly post-commit — preserving the old synchronous fan-out timing — while the
	// per-recipient Outbox:SendToSingleRecipient deliveries it enqueues stay background (256 below).
	case "Outbox-Publish":
		task.Priority = 16

	case "SendSearchResult":
		task.Priority = 16

	case "SendSearchResult-SearchQuery":
		task.Priority = 16

	case "ReceiveActivityPub-Move":
		task.Priority = 16

	case "SendWebPushNotification":
		task.Priority = 16

	// A human pressed Sync Now and is watching the settings screen for the answer.  Its twin,
	// TaskSyncStreamSource, is the webhook's background fan-out at 256 below.  Note that the
	// priority alone does not make this immediate: PublishSyncTaskNow also omits the signature,
	// because turbine will not run a signed task from memory at any priority.
	case service.TaskSyncStreamSourceNow:
		task.Priority = 16

	// (32) User-Affecting Tasks That Should Complete Very Quickly

	///////////////////////////////////////////////////
	// Tasks below this line are ALWAYS written to the
	// database, and are NOT executed immediately

	// (64) User Facing but Low Priority Tasks
	case "Geocode":
		task.Priority = 64

	case "ImportItem":
		task.Priority = 64

	case "ReceiveActivityPub-Add":
		task.Priority = 64

	// (256) Background Notifications
	// The mailing-list sync is one HTTP call per follower against the User's own Mailchimp
	// quota. It is never user-facing, and a minute late costs nothing.
	case service.MailingListAddMember, service.MailingListRemoveMember:
		task.Priority = 256

	case "MakeStreamArchive":
		task.Priority = 256

	case "Outbox:SendToAllRecipients":
		task.Priority = 256

	case "Outbox:SendToSingleRecipient":
		task.Priority = 256

	case "syndication.create", "syndication.update", "syndication.delete":
		task.Priority = 256

	// SyncStreamSource is deliberately NOT in the <= 32 band.  Its trigger is an unauthenticated
	// webhook that fans out to every record sharing a token, so an immediate priority would let
	// one ping turn into a burst of outbound requests with nothing in between.  The interactive
	// twin, TaskSyncStreamSourceNow, is at 16 above; one caller decides which name to publish.
	case service.TaskSyncStreamSource:
		task.Priority = 256

	// (512) System Tasks that should happen mostly on time
	case "DeleteStream":
		task.Priority = 512

	case "PollFollowing-Index":
		task.Priority = 512

	case "PollFollowing-Record":
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

	case "ReceiveActivityPub-Delete":
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

	case "PurgeNotifications":
		task.Priority = 1024

	case "RecycleDomain":
		task.Priority = 1024
	}

	return nil
}
