package gofed

import "context"

// MaxInboxForwardingRecursionDepth determines how deep to search within an activity's
// historical chain to determine if inbox forwarding needs to occur. After reaching this
// depth, it is assumed that peers deeper than that conversational depth are no longer
// candidates for triggering the inbox forwarding logic.
//
// Zero or negative numbers indicate recurring infinitely, which can result in your
// application being manipulated by malicious peers. Do not return a value of zero nor
// a negative number.
func (fed Federating) MaxInboxForwardingRecursionDepth(c context.Context) int {
	// TODO: LOW: re-evaluate this guess once ActivityPub is running.
	return 2
}
