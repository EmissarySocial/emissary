package consumer

import (
	"time"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
)

func Scheduler(serverFactory ServerFactory) queue.Result {

	const location = "consumer.ScheduleDaily"
	log.Trace().Str("location", location).Msg("Initializing Scheduler...")

	// Schedule the next batch of daily tasks
	if err := scheduler_MakeDailyTasks(serverFactory); err != nil {
		return queue.Error(err)
	}

	// Schedule the next batch of hourly tasks
	if err := scheduler_MakeHourlyTasks(serverFactory); err != nil {
		return queue.Error(err)
	}

	return queue.Success()
}

// creates new DAILY tasks beginning tomorrow
func scheduler_MakeDailyTasks(serverFactory ServerFactory) error {

	const location = "consumer.scheduleDaily_MakeDailyTasks"

	// Schedule daily tasks for the next two days
	for nextDay := 1; nextDay <= 2; nextDay++ {

		// Calculate the start date
		startTime := time.Now().
			AddDate(0, 0, nextDay).
			Truncate(24 * time.Hour)

		// Create a new task
		task := queue.NewTask(
			"ScheduleDaily",
			mapof.NewAny(),
			queue.WithStartTime(startTime),
			queue.WithSignature("DAILY:"+startTime.Format("2006-01-02")),
		)

		// Publish the task to the queue
		if err := serverFactory.Queue().Publish(task); err != nil {
			return derp.Wrap(err, location, "Unable to publish tomorrow's daily task")
		}
	}

	// Magnificent.
	return nil
}

// creates 24 new HOURLY tasks beginning at the top of each hour
func scheduler_MakeHourlyTasks(serverFactory ServerFactory) error {

	const location = "consumer.scheduleDaily_MakeHourlyTasks"

	for nextHour := 1; nextHour < 48; nextHour++ {

		// Calculate the start time
		startTime := time.Now().
			Add(time.Duration(nextHour * int(time.Hour))).
			Truncate(time.Hour)

		// Create a new task
		task := queue.NewTask(
			"ScheduleHourly",
			mapof.NewAny(),
			queue.WithStartTime(startTime),
			queue.WithSignature("HOURLY:"+startTime.Format("2006-01-02 15:00")),
		)

		// Publish the task to the queue
		if err := serverFactory.Queue().Publish(task); err != nil {
			return derp.Wrap(err, location, "Unable to publish hourly task")
		}
	}

	// Glorious.
	return nil
}
