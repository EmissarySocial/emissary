// Package counter tallies how many times each key has been seen.
//
// A Counter is a thin wrapper over rosetta's mapof.Int that adds the one operation callers
// actually want -- increment this key by one -- so that counting does not require the
// present/absent dance at every call site.
//
// Counters are not safe for concurrent use.  Build one inside a single request or task.
package counter
