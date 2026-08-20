package server

import (
	"context"
	"time"
)

// timeoutContext returns a background context that cancels itself after the provided number of seconds
func timeoutContext(seconds int) (context.Context, context.CancelFunc) {

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(seconds)*time.Second)

	// Return the context and cancel function
	return ctx, cancel
}
