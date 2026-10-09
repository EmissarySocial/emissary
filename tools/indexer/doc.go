// Package indexer makes a MongoDB collection's indexes match a declared set.
//
// Sync compares the indexes a collection actually has against the IndexSet the code declares,
// then creates what is missing and drops what is no longer named.  Emissary calls it at
// startup so index definitions live beside the queries that need them rather than in a
// migration nobody re-runs.
//
// Comparison is the delicate part.  MongoDB reads an index spec back in a normalized form,
// so a spec written one way and stored another looks like a mismatch, and Sync would drop
// and rebuild the index on every boot.  The helpers here normalize both sides first; prefer
// bson.M over bson.D at call sites to stay on the tested path.
package indexer
