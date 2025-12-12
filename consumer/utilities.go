package consumer

import (
	"time"

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

/* Unused function
func timeoutContext(seconds int) (context.Context, context.CancelFunc) {
	return context.WithTimeout(
		context.Background(),
		time.Duration(seconds)*time.Second,
	)
}
*/
