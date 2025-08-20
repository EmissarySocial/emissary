package server

import (
	"context"
	"time"
)

func timeoutContext(seconds int) (context.Context, context.CancelFunc) {

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(seconds)*time.Second)

	// Return the context and cancel function
	return ctx, cancel
}
