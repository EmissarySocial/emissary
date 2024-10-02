package queue

import (
	"context"
	"time"
)

// timeoutContext returns a context that times out after timeoutSeconds seconds
func timeoutContext(timeoutSeconds int) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
}

// backoff calculates the exponential backoff time for a retry
func backoff(retryCount int) time.Duration {
	return time.Duration(2^retryCount) * time.Minute
}
