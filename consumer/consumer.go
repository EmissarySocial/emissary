package consumer

import (
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

	case "CrawlActivityStreams":
		return WithSession(consumer.serverFactory, args, CrawlActivityStreams)

	case "CreateWebSubFollower":
		return WithSession(consumer.serverFactory, args, CreateWebSubFollower)

	case "CountRelatedDocuments":
		return WithFactory(consumer.serverFactory, args, CountRelatedDocuments)

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

	case "LoadActivityStream":
		return WithSession(consumer.serverFactory, args, LoadActivityStream)

	case "MakeStreamArchive":
		return WithStream(consumer.serverFactory, args, MakeStreamArchive)

	case "PollFollowing":
		return WithSession(consumer.serverFactory, args, PollFollowing)

	case "ProcessMedia":
		return WithSession(consumer.serverFactory, args, ProcessMedia)

	case "PurgeActivityStreamCache":
		return PurgeActivityStreamCache(consumer.serverFactory)

	case "PurgeErrors":
		return PurgeErrors(consumer.serverFactory)

	case "PurgeDomeLog":
		return PurgeDomeLog(consumer.serverFactory)

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
