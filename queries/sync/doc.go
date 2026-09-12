// Package sync declares the MongoDB indexes for every collection in a Domain database.
//
// Each file here holds one function per collection, returning the index set that collection
// needs; tools/indexer compares that set against what the database actually has and
// reconciles the difference at startup.
//
// Index definitions live here, next to each other, rather than in the upgrade slots -- an
// upgrade runs once per install, but an index set must be re-asserted on every boot so a new
// server, a restored backup, and a long-running one all end up identical.
package sync
