package ascache

import (
	"context"
	"time"

	"github.com/benpate/hannibal/streams"
)

func timeoutContext(seconds int) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(seconds)*time.Second)
}

func asValue(document streams.Document) Value {

	result := NewValue()
	result.URLs = append(result.URLs, document.ID())
	result.Object = document.Map()
	result.HTTPHeader = document.HTTPHeader()
	result.Metadata = document.Metadata

	return result
}
