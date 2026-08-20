package asnormalizer

import "github.com/benpate/hannibal/streams"

// Tombstone normalizes a Tombstone document into a plain map
func Tombstone(rootClient streams.Client, document streams.Document) map[string]any {
	return document.Map()
}
