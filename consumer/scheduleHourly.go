package consumer

import (
	"hash/fnv"

	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
)

// ScheduleHourly queues the next hour's worth of scheduled tasks for every Domain
func ScheduleHourly(serverFactory ServerFactory) queue.Result {

	const location = "consumer.ScheduleHourly"
	log.Trace().Str("location", location).Msg("Running Hourly Tasks...")

	q := serverFactory.Queue()

	// Hourly tasks for each domain
	for factory := range serverFactory.RangeDomains() {

		// Email the Users who have left notifications unread for longer than they chose
		q.NewTask("SendNotificationEmails", mapof.Any{"hostname": factory.Hostname()})

		// Reshuffle the search results.  Each Domain starts at its own minute, so they do not all
		// write to the shared database at once.
		q.NewTask("Shuffle", mapof.Any{"hostname": factory.Hostname()}, queue.WithDelaySeconds(hourlyOffsetSeconds(factory.Hostname())))

		// Schedule "PollFollowing-Index" tasks every four hours, starting at 1am.
		if isHour(4, 1) {
			q.NewTask("PollFollowing-Index", mapof.Any{"hostname": factory.Hostname()})
		}
	}

	return queue.Success()
}

// hourlyOffsetWindow is the part of each hour, in seconds, that hourly tasks are spread across.
// The last ten minutes stay clear, so a slow run ends before the next hour's begins.
const hourlyOffsetWindow = 50 * 60

// hourlyOffsetSeconds returns how long after the top of the hour a Domain's spread-out hourly
// tasks begin.  It is the same for a hostname every hour, and on every server.
func hourlyOffsetSeconds(hostname string) int {

	hash := fnv.New32a()
	_, _ = hash.Write([]byte(hostname)) // fnv's Write never returns an error

	return int(hash.Sum32() % hourlyOffsetWindow)
}
