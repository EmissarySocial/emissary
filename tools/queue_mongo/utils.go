package queue_mongo

import (
	"context"
	"time"
)

// timeoutContext returns a context that times out after timeoutSeconds seconds
func timeoutContext(timeoutSeconds int) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
}
