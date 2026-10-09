# queries

This package contains the custom database queries that don't fit the standard CRUD operations of the [service](../service/) layer — aggregates, counters, and other specialized reads and writes against MongoDB. The [sync](sync/) subdirectory holds queries that reconcile related collections, and [upgrades](upgrades/) holds the per-version data-migration scripts that bring an existing database up to the current schema.

The `Watch*` functions follow MongoDB change streams for as long as their domain is running. `WatchStreams`, `WatchUsers`, and `WatchImports` tell open browser pages to refresh, and `WatchDomain` keeps every server's cached copy of the Domain record current, so a setting saved on one server reaches the others. All four run inside the supervisor in [watch.go](watch.go), which reopens a stream whenever MongoDB closes it. Change streams need a replica set; on a standalone server the watchers stop quietly.

See the [project README](../README.md) for the big picture.
