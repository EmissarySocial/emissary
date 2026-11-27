package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/hannibal/delivery"
	"github.com/benpate/turbine/queue"
)

// PreProcessor gives a task the priority that the table below lists for its name, unless its
// publisher already chose one.  Consumer.OnPublish runs it on every task that enters the queue.
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
	case "Outbox-Publish", delivery.TaskFanOut:
		task.Priority = 16

	// The outbox task decides what a Stream's followers are sent, so it runs promptly after the
	// commit, as Outbox-Publish does.  The deliveries it queues stay in the background (256 below).
	case service.StreamOutboxTaskName:
		task.Priority = 16

	case service.TaskSendDeleteQuoteAuthorization:
		task.Priority = 16

	case "SendSearchResult":
		task.Priority = 16

	case "SendSearchResult-SearchQuery":
		task.Priority = 16

	case "ReceiveActivityPub-Move":
		task.Priority = 16

	case "SendWebPushNotification":
		task.Priority = 16

	// Every StreamSource sync runs here, whether Sync Now or a webhook asked for it.  The priority
	// alone does not make one immediate: turbine will not run a SIGNED task from memory, so a
	// webhook's deduplicated task still waits for the storage poller.
	case service.TaskSyncStreamSource:
		task.Priority = 16

	// A file linked from a synchronized source.  Signed per attachment, so it waits for the
	// storage poller as a webhook sync does.
	case service.TaskSyncStreamSourceAttachment:
		task.Priority = 16

	// (32) User-Affecting Tasks That Should Complete Very Quickly

	///////////////////////////////////////////////////
	// Tasks below this line are ALWAYS written to the
	// database, and are NOT executed immediately

	// (64) User Facing but Low Priority Tasks
	case service.TaskGeocode:
		task.Priority = 64

	// A post's preview card appears on a later load, as on Mastodon, so the read can wait
	case service.TaskFetchLinkPreview:
		task.Priority = 64

	case "ImportItem":
		task.Priority = 64

	case "ReceiveActivityPub-Add":
		task.Priority = 64

	case "RepairStripeConnect", "ReconcileStripeSubscriptions":
		task.Priority = 64

	// (256) Background Notifications
	// The mailing-list sync is one HTTP call per follower against the User's own Mailchimp
	// quota. It is never user-facing, and a minute late costs nothing.
	case service.MailingListAddMember, service.MailingListRemoveMember:
		task.Priority = 256

	case "MakeStreamArchive":
		task.Priority = 256

	// A one-time repair that sends nothing, so it waits behind everything a person is waiting for
	case service.StreamFillFingerprintsTaskName:
		task.Priority = 256

	case "Outbox:SendToAllRecipients":
		task.Priority = 256

	case "Outbox:SendToSingleRecipient":
		task.Priority = 256

	// RULE: Deliveries run below user-facing work, so that a large fan-out cannot delay it
	case delivery.TaskSend, service.TaskOutbox2Email:
		task.Priority = 256

	case "syndication.create", "syndication.update", "syndication.delete":
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

	case TaskProbeDormantHosts:
		task.Priority = 1024

	case "PurgeNotifications":
		task.Priority = 1024

	case "PurgeStagedUploads":
		task.Priority = 1024

	case "SendNotificationEmails":
		task.Priority = 1024

	case "RecycleDomain":
		task.Priority = 1024
	}

	return nil
}
