# queries

This package contains the custom database queries that don't fit the standard CRUD operations of the [service](../service/) layer — aggregates, counters, and other specialized reads and writes against MongoDB. The [sync](sync/) subdirectory holds queries that reconcile related collections, and [upgrades](upgrades/) holds the per-version data-migration scripts that bring an existing database up to the current schema.

See the [project README](../README.md) for the big picture.
