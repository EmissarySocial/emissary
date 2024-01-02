package asnormalizer

import "github.com/benpate/hannibal/streams"

// Context computes the document context (not the @context)
// Using either the context field: https://www.w3.org/ns/activitystreams#context
// or the (deprecated) ostatus:conversation field
func Context(document streams.Document) string {

	if context := document.Context(); context != "" {
		return context
	}

	if conversation := document.Get("conversation").String(); conversation != "" {
		return conversation
	}

	return ""
}
