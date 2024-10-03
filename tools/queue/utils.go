package queue

import (
	"time"
)

// backoff calculates the exponential backoff time for a retry
func backoff(retryCount int) time.Duration {
	return time.Duration(2^retryCount) * time.Minute
}
