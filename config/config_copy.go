package config

import (
	"maps"
	"slices"

	"github.com/benpate/rosetta/sliceof"
)

// Copy returns an independent copy of this Config.
//
// RULE: Every Config that leaves the server factory MUST be copied, not merely assigned.  A plain
// struct assignment copies the header of each map and slice but shares the backing storage, so a
// caller that edits (say) `AttachmentCache["location"]` reaches straight through into the running
// server's live configuration -- concurrently with the reload goroutine and every request that
// reads it.  The setup console's form handlers do exactly that: they take a Config, apply posted
// values to it, and save.  See [[config-shallow-copy-map-aliasing]].
//
// The copy is one level deep, which covers every reference type this struct actually holds:
// Domain is all scalars, and the map values are the scalars that come back out of JSON and BSON.
func (config Config) Copy() Config {

	result := config

	result.Domains = slices.Clone(config.Domains)
	result.Templates = copyMapSlice(config.Templates)
	result.AttachmentOriginals = maps.Clone(config.AttachmentOriginals)
	result.AttachmentCache = maps.Clone(config.AttachmentCache)
	result.ExportCache = maps.Clone(config.ExportCache)
	result.Certificates = maps.Clone(config.Certificates)
	result.ActivityPubCache = maps.Clone(config.ActivityPubCache)
	result.Loggers = copyMapSlice(config.Loggers)

	return result
}

// copyMapSlice clones a slice of maps, and each map inside it, so neither the slice nor any of its
// entries stays connected to the original.
func copyMapSlice[V any, T ~map[string]V](original sliceof.Object[T]) sliceof.Object[T] {

	if original == nil {
		return nil
	}

	result := make(sliceof.Object[T], len(original))

	for index, value := range original {
		result[index] = maps.Clone(value)
	}

	return result
}
