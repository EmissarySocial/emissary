// Package floodgatemongo keeps Floodgate's host records in MongoDB.
//
// Floodgate decides whether this server may deliver to a remote host, from a record of how
// recent deliveries to that host went. Every Emissary server in a cluster must see the same
// record, so the records live in the common database, in one collection keyed by hostname.
//
// Each method is a single MongoDB operation on one host's record: an update with operators, or
// a find-and-update. Nothing here reads a record and writes it back, so two servers that report
// on the same host at the same moment are both counted.
package floodgatemongo
