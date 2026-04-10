package consumer

import (
	"time"

	"github.com/benpate/derp"
	dt "github.com/benpate/domain"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func objectID(original string) primitive.ObjectID {
	result, _ := primitive.ObjectIDFromHex(original)
	return result
}

// isHour returns TRUE if the current hour matches the modulo and startHour.
// For example, isHour(2, 0) returns TRUE every two hours from midnight, 2am, 4am, etc.
// and isHour(6, 1) returns TRUE every six hours starting at 1am, 7am, 1pm, 7pm
func isHour(modulo int, startHour int) bool {
	currentHour := time.Now().Hour()
	return startHour == (currentHour % modulo)
}

// getHostnameFromArgs attempts to extract a hostname from the provided args map.
func getHostnameFromArgs(args mapof.Any) string {

	// If the args include a "host" argument, then use that first
	if host := args.GetString("host"); host != "" {
		return dt.Hostname(host)
	}

	// If an "actor" argument is provided, then use that to determine the hostname
	if actorURL := args.GetString("actor"); actorURL != "" {
		return dt.Hostname(actorURL)
	}

	return ""
}

// requeue wraps errors with intelligent retry logic
func requeue(err error) queue.Result {

	// If there is no error, then return success
	if err == nil {
		return queue.Success()
	}

	// Retry HTTP 429 (Too Many Requests) errors
	if isTooMany, delay := derp.IsTooManyRequests(err); isTooMany {
		return queue.Requeue(delay)
	}

	// Client Errors (400) can't be retried. Fail the whole task
	if derp.IsClientError(err) {
		return queue.Failure(err)
	}

	// Server Errors (500) can be retried. Report a retryable error.
	return queue.Error(err)
}

/* Unused function
func timeoutContext(seconds int) (context.Context, context.CancelFunc) {
	return context.WithTimeout(
		context.Background(),
		time.Duration(seconds)*time.Second,
	)
}
*/
