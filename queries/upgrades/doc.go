// Package upgrades holds the ordered data migrations for a Domain database.
//
// Each VersionN function migrates a Domain from schema version N-1 to N.  queries/upgrade.go
// holds them in a slice whose INDEX is the databaseVersion recorded on each Domain record,
// and a server that has already recorded version N will never run slot N again.
//
// So a slot number is permanent.  Changing what an existing slot does skips the migration on
// every deployed server while running it on new ones, which is silent and unrecoverable.
// Only ever append; to correct a bad migration, add the correction as the next slot.
//
// ForEachRecord is the shared helper for slots that must rewrite documents one at a time.
package upgrades
