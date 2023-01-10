package gofed

import "context"

/*
This method determines how deep to search within collections owned by peers when they are targeted to receive a delivery. After reaching this depth, it is assumed that peers deeper than that are no longer interested in receiving messages. A positive number must be returned.

Zero or negative numbers indicate recurring infinitely, which can result in your application being manipulated by malicious peers. Do not return a value of zero nor a negative number.
*/
func (fed Federating) MaxDeliveryRecursionDepth(c context.Context) int {
	// TODO: CRITICAL: Do This
	return 0
}
