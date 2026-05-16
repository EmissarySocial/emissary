package asnormalizer

import "github.com/benpate/hannibal/streams"

func Tombstone(rootClient streams.Client, document streams.Document) map[string]any {
	return document.Map()
}
