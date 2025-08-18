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

	case
		"CrawlActivityStreams",
		"LoadActivityStream":
		return WithFactory(consumer.serverFactory, args, CrawlActivityStreams)

	case "CreateWebSubFollower":
		return WithFactory(consumer.serverFactory, args, CreateWebSubFollower)

	case "DeleteEmptySearchQuery":
		return WithFactory(consumer.serverFactory, args, DeleteEmptySearchQuery)

	case "Geocode":
		return WithStream(consumer.serverFactory, args, Geocode)

	case "IndexAllStreams":
		return WithFactory(consumer.serverFactory, args, IndexAllStreams)

	case "IndexAllUsers":
		return WithFactory(consumer.serverFactory, args, IndexAllUsers)

	case "MakeStreamArchive":
		return WithStream(consumer.serverFactory, args, MakeStreamArchive)

	case "PollFollowing":
		return WithFactory(consumer.serverFactory, args, PollFollowing)

	case "ProcessMedia":
		return WithFactory(consumer.serverFactory, args, ProcessMedia)

	case "PurgeActivityStreamCache":
		return PurgeActivityStreamCache(consumer.serverFactory)

	case "PurgeErrors":
		return PurgeErrors(consumer.serverFactory)

	case "PurgeDomeLog":
		return PurgeDomeLog(consumer.serverFactory)

	case "ReceiveWebMention":
		return WithFactory(consumer.serverFactory, args, ReceiveWebMention)

	case "RecycleDomain":
		return WithFactory(consumer.serverFactory, args, RecycleDomain)

	case "Scheduler":
		return Scheduler(consumer.serverFactory)

	case "ScheduleDaily":
		return ScheduleDaily(consumer.serverFactory)

	case "ScheduleHourly":
		return ScheduleHourly(consumer.serverFactory)

	case "SendActivityPubMessage":
		return WithFactory(consumer.serverFactory, args, SendActivityPubMessage)

	case "SendSearchResults":
		return WithFactory(consumer.serverFactory, args, SendSearchResults)

	case "SendSearchResults-Query":
		return WithFactory(consumer.serverFactory, args, SendSearchResults_Query)

	case "SendSearchResults-Global":
		return WithFactory(consumer.serverFactory, args, SendSearchResults_Global)

	case "SendWebMention":
		return SendWebMention(args)

	case "SendWebSubMessage":
		return SendWebSubMessage(args)

	case "Shuffle":
		return WithFactory(consumer.serverFactory, args, Shuffle)

	case "syndication.create", "syndication.update", "syndication.delete":
		return StreamSyndicate(name, args)

	case "UpdateContext":
		return UpdateContext(consumer.serverFactory, args)
	}

	return queue.Ignored()
}
