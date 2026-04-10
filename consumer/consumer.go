package consumer

import (
	"github.com/benpate/hannibal/sender"
	"github.com/benpate/turbine/queue"
)

// Consumer is the primary queue consumer for Emissary.  It handles background tasks that are triggered by the queue.
type Consumer struct {
	serverFactory ServerFactory
}

// New returns a fully initialized Consumer object
func New(serverFactory ServerFactory) Consumer {
	return Consumer{
		serverFactory: serverFactory,
	}
}

// Run is the actual consumer function that is called by the queue.
// It receives a task name and a map of arguments, and returns a boolean success value and an error.
func (consumer Consumer) Run(name string, args map[string]any) queue.Result {

	switch name {

	case "AddToContext":
		return WithSession(consumer.serverFactory, args, AddToContext)

	case "ConnectPushService":
		return WithFollowing(consumer.serverFactory, args, ConnectPushService)

	case "CrawlContext":
		return WithFactory(consumer.serverFactory, args, CrawlContext)

	case "CrawlUpReplyTree":
		return WithFactory(consumer.serverFactory, args, CrawlUpReplyTree)

	case "CreateWebSubFollower":
		return WithSession(consumer.serverFactory, args, CreateWebSubFollower)

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

	case "MakeStreamArchive":
		return WithStream(consumer.serverFactory, args, MakeStreamArchive)

	case "MoveUser":
		return WithUser(consumer.serverFactory, args, MoveUser)

	case sender.OutboxSendToAllRecipients:
		return WithSender(consumer.serverFactory, args, SendToAllRecipients)

	case sender.OutboxSendToSingleRecipient:
		return WithSender(consumer.serverFactory, args, SendToSingleRecipient)

	case "PollFollowing-Index":
		return WithSession(consumer.serverFactory, args, PollFollowing_Index)

	case "PollFollowing-Record":
		return WithFollowing(consumer.serverFactory, args, PollFollowing_Record)

	case "PurgeActivityStreamCache":
		return PurgeActivityStreamCache(consumer.serverFactory)

	case "PurgeErrors":
		return PurgeErrors(consumer.serverFactory)

	case "PurgeDomeLog":
		return PurgeDomeLog(consumer.serverFactory)

	case "PurgeImports":
		return WithSession(consumer.serverFactory, args, PurgeImports)

	case "ReceiveActivityPub-Add":
		return WithSession(consumer.serverFactory, args, ReceiveActivityPubAdd)

	case "ReceiveActivityPub-Delete":
		return WithSession(consumer.serverFactory, args, ReceiveActivityPubDelete)

	case "ReceiveActivityPub-Move":
		return WithSession(consumer.serverFactory, args, ReceiveActivityPubMove)

	case "ReceiveWebMention":
		return WithSession(consumer.serverFactory, args, ReceiveWebMention)

	case "RecycleDomain":
		return WithSession(consumer.serverFactory, args, RecycleDomain)

	case "ReindexActivityStream":
		return WithFactory(consumer.serverFactory, args, ReindexActivityStream)

	case "Scheduler":
		return Scheduler(consumer.serverFactory)

	case "ScheduleStartup":
		return ScheduleStartup(consumer.serverFactory)

	case "ScheduleDaily":
		return ScheduleDaily(consumer.serverFactory)

	case "ScheduleHourly":
		return ScheduleHourly(consumer.serverFactory)

	// TODO: This should be merged into Outbox:SendToAllRecipients
	case "SendActivityPubMessage":
		return WithSession(consumer.serverFactory, args, SendActivityPubMessage)

	case "SendSearchResult":
		return WithSession(consumer.serverFactory, args, SendSearchResult)

	case "SendSearchResult-SearchQuery":
		return WithSession(consumer.serverFactory, args, SendSearchResult_SearchQuery)

	case "SendWebMention":
		return SendWebMention(args)

	case "SendWebSubMessage":
		return SendWebSubMessage(args)

	case "Shuffle":
		return WithSession(consumer.serverFactory, args, Shuffle)

	case "syndication.create", "syndication.update", "syndication.delete":
		return StreamSyndicate(name, args)
	}

	return queue.Ignored()
}
