package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/hannibal/sender"
	"github.com/benpate/turbine/queue"
)

// Consumer is the primary queue consumer for Emissary.  It handles background tasks that are triggered by the queue.
type Consumer struct {
	serverFactory ServerFactory
}

// RULE: Every method of queue.Consumer is required, so a hook whose name or signature drifted
// would fail to compile here rather than quietly never being called.
var _ queue.Consumer = Consumer{}

// New returns a fully initialized Consumer object
func New(serverFactory ServerFactory) Consumer {
	return Consumer{
		serverFactory: serverFactory,
	}
}

// Run executes a single attempt of a background task.
// Implements the queue.Consumer interface.
func (consumer Consumer) Run(task queue.Task) queue.Result {

	// Unpacked in the switch itself so that the task table below reads the way it did before
	// the queue passed the whole Task
	switch name, args := task.Name, task.Arguments; name {

	case "AddToCollection":
		return WithSession(consumer.serverFactory, args, AddToCollection)

	case "ConnectPushService":
		return WithFollowing(consumer.serverFactory, args, ConnectPushService)

	case "CrawlContext":
		return WithFactory(consumer.serverFactory, args, CrawlContext)

	case "CrawlUpReplyTree":
		return WithFactory(consumer.serverFactory, args, CrawlUpReplyTree)

	case "CrawlDownReplyTree":
		return WithFactory(consumer.serverFactory, args, CrawlDownReplyTree)

	case "DeleteEmptySearchQuery":
		return WithSession(consumer.serverFactory, args, DeleteEmptySearchQuery)

	case "DeleteStream":
		return WithSession(consumer.serverFactory, args, DeleteStream)

	case "Geocode":
		return WithStream(consumer.serverFactory, args, Geocode)

	case "ImportStartup":
		return WithImport(consumer.serverFactory, args, ImportStartup)

	case "ImportItems":
		return WithImport(consumer.serverFactory, args, ImportItems)

	case service.MailingListAddMember:
		return WithSession(consumer.serverFactory, args, MailingListAddMember)

	case service.MailingListRemoveMember:
		return WithSession(consumer.serverFactory, args, MailingListRemoveMember)

	case "MakeStreamArchive":
		return WithStream(consumer.serverFactory, args, MakeStreamArchive)

	case "MoveUser":
		return WithUser(consumer.serverFactory, args, MoveUser)

	case sender.OutboxSendToAllRecipients:
		return WithSender(consumer.serverFactory, args, SendToAllRecipients)

	case sender.OutboxSendToSingleRecipient:
		return WithSender(consumer.serverFactory, args, SendToSingleRecipient)

	case "Outbox-Publish":
		return WithSession(consumer.serverFactory, args, OutboxPublish)

	case "PollFollowing-Index":
		return WithSession(consumer.serverFactory, args, PollFollowing_Index)

	case "PollFollowing-Record":
		return WithFollowing(consumer.serverFactory, args, PollFollowing_Record)

	case "PublishRealtimeMessage":
		return WithFactory(consumer.serverFactory, args, PublishRealtimeMessage)

	case "PurgeActivityStreamCache":
		return PurgeActivityStreamCache(consumer.serverFactory)

	case "PurgeErrors":
		return PurgeErrors(consumer.serverFactory)

	case "PurgeDomeLog":
		return PurgeDomeLog(consumer.serverFactory)

	case "PurgeImports":
		return WithSession(consumer.serverFactory, args, PurgeImports)

	case "PurgeNotifications":
		return WithSession(consumer.serverFactory, args, PurgeNotifications)

	case "Rule-Cleanup":
		return WithSession(consumer.serverFactory, args, RuleCleanup)

	case "ReceiveActivityPub-Add":
		return WithSession(consumer.serverFactory, args, ReceiveActivityPubAdd)

	case "ReceiveActivityPub-Delete":
		return WithSession(consumer.serverFactory, args, ReceiveActivityPubDelete)

	case "ReceiveActivityPub-Move":
		return WithSession(consumer.serverFactory, args, ReceiveActivityPubMove)

	case "ReconcileStripeSubscriptions":
		return WithFactory(consumer.serverFactory, args, ReconcileStripeSubscriptions)

	case "RecycleDomain":
		return WithSession(consumer.serverFactory, args, RecycleDomain)

	case "ReindexActivityStream":
		return WithFactory(consumer.serverFactory, args, ReindexActivityStream)

	case "RepairStripeConnect":
		return WithSession(consumer.serverFactory, args, RepairStripeConnect)

	case "Scheduler":
		return Scheduler(consumer.serverFactory)

	case "ScheduleStartup":
		return ScheduleStartup(consumer.serverFactory)

	case "ScheduleDaily":
		return ScheduleDaily(consumer.serverFactory)

	case "ScheduleHourly":
		return ScheduleHourly(consumer.serverFactory)

	case "SendSearchResult":
		return WithSession(consumer.serverFactory, args, SendSearchResult)

	case "SendSearchResult-SearchQuery":
		return WithSession(consumer.serverFactory, args, SendSearchResult_SearchQuery)

	case "SendWebPushNotification":
		return WithSession(consumer.serverFactory, args, SendWebPushNotification)

	case "Shuffle":
		return WithSession(consumer.serverFactory, args, Shuffle)

	// Both synchronization tasks run the SAME handler.  They are named apart only so the
	// priority table can tell a human pressing Sync Now from a webhook's background fan-out.
	case service.TaskSyncStreamSource, service.TaskSyncStreamSourceNow:
		return WithSession(consumer.serverFactory, args, SyncStreamSource)

	case "syndication.create", "syndication.update", "syndication.delete":
		return StreamSyndicate(name, args)
	}

	return queue.Ignored()
}

// OnPublish is called for every task on its way onto the queue.
// Implements the queue.Consumer interface.
func (consumer Consumer) OnPublish(task *queue.Task) error {
	// No task changes itself on the way onto the queue yet.  This is where the priority table in
	// PreProcessor belongs, once someone decides to turn it on -- see preprocessor.go.
	return nil
}

// OnSuccess is called after an attempt that succeeded.
// Implements the queue.Consumer interface.
func (consumer Consumer) OnSuccess(task queue.Task) error {

	if service.IsSyncStreamSourceTask(task.Name) {
		return syncStreamSourceSucceeded(consumer.serverFactory, task.Arguments)
	}

	// No other task reports its successes yet.
	return nil
}

// OnError is called after an attempt that failed and WILL be tried again.
// Implements the queue.Consumer interface.
func (consumer Consumer) OnError(task queue.Task, err error) error {

	if service.IsSyncStreamSourceTask(task.Name) {
		return syncStreamSourceRetrying(consumer.serverFactory, task.Arguments, err)
	}

	// No other task reports its retries yet.
	return nil
}

// OnFailure is called when a task is abandoned and will NOT be tried again.
// Implements the queue.Consumer interface.
func (consumer Consumer) OnFailure(task queue.Task, err error) error {

	if service.IsSyncStreamSourceTask(task.Name) {
		return syncStreamSourceFailed(consumer.serverFactory, task.Arguments, err)
	}

	// No other task reports its abandonment yet.
	return nil
}
