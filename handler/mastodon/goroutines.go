package mastodon

import (
	"fmt"

	"github.com/benpate/derp"
)

// recoverGoroutine reports a panic in a goroutine this package started, so that it fails one result
// instead of ending the process. Defer it first in every such goroutine.
func recoverGoroutine(location string) {

	// Echo's Recover middleware guards only the request goroutine, so these need their own
	if recovered := recover(); recovered != nil {
		derp.Report(derp.Internal(location, "Recovered from a panic in a goroutine", fmt.Sprint(recovered)))
	}
}
