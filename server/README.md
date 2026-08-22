# server

This package wires up the running Emissary server. Its `Factory` owns the server-level services (those shared across every domain) and produces an individual domain factory for each domain the server hosts — the seam between one process and the many domains it serves. This is where server-wide configuration is turned into live, per-domain machinery.

## Configuration reloads

A configuration reload arrives on a background goroutine while every in-flight HTTP request is still reading what the last one produced. That single fact shapes everything below.

Everything a reload replaces — the configuration itself, the mounted filesystems, the common (ActivityPub Cache) database, the task queue, the client-IP strategy — lives in a single immutable `wiring` value, published with one atomic store. See [wiring.go](wiring.go).

- **Readers take no lock at all.** `currentWiring()` is an atomic load. A reader gets a whole generation, so it can never see the new queue beside the old database, or one configuration's database URI beside another's name.
- **Published wiring is immutable.** A reader may hold its pointer indefinitely, on any goroutine. Change something by publishing a *new* generation, never by writing through a pointer you loaded. For the configuration — the one field holding maps and slices — the writers `setConfigLocked` and `withConfigLocked` deep-copy (`config.Config.Copy`) before editing, so no generation ever shares map storage with another. `Config()` deep-copies on the way *out* for the same reason: callers (the setup console's form handlers above all) treat the returned value as scratch space and edit it in place.
- **Writers hold `reloadLock` for the whole decide-then-publish sequence.** It serializes writers against each other and is never taken by a reader, so it can be held across the slow parts — opening a mongo client, a five-second ping, a storage write, synchronizing every shared index — without stalling a single request. The `Locked` suffix is the convention: `rewireLocked`/`setConfigLocked`/`withConfigLocked`/`updateConfigLocked` assume you hold it; `rewire` and `UpdateConfig` take it for you.

The shape of every refresh step is *decide → do the slow work unlocked → publish → disconnect the old object unlocked*. `refreshCommonDatabase` is the worked example — and the **one** connect path for every mode: with `verify` on (setup console) the new connection must answer a Ping before it is published and rolls back to "not connected" if it doesn't; with `verify` off (live server) the connection is published as opened, because live mode verifies by using it.

**Saves are compare-and-swap, and intent decides who retries.** `Storage.Write` rejects a save built from a stale revision with a 409. `updateConfigLocked` (the setup console's whole-form save) surfaces the conflict to the human — replaying their edit over someone else's change would be the silent overwrite the revision exists to prevent — and publishes locally only *after* a successful write, so a rejected save leaves the node unchanged. `mutateConfigLocked` (adding/removing domains) *rebases and retries*: its mutations state an intent keyed by DomainID that is safe to re-apply, and the rebase reads from **storage**, not this node's wiring — a conflict means the wiring is stale by definition.

**A rejected reload keeps the last-known-good configuration.** `readConfig` validates everything that can fail *before* publishing anything, and returns an error having applied nothing. The *first* configuration failing is fatal — `NewFactory` returns the error and `main` refuses to start, since a server that never had a working configuration has nothing to keep serving. A configuration failing *later* is reported loudly and the node keeps serving on its previous configuration — the alternative (exiting, as this code once did) let one bad save take down every node in a cluster simultaneously, then crash-loop them all against the same stored document. Nothing in `config/` or `server/` calls `os.Exit`; `main` owns every exit.

**What this does not buy:** immutability stops *tearing*, not *use-after-swap*. A handler that calls `Queue()` and then publishes can still hand a task to a queue that was stopped a microsecond later. The mitigations are the `queueReady`/`queueDatabase` guards that make rebuilds rare, and the standing rule that domain factories read *through* the factory on every use instead of capturing handles.

**If you add a field to `factoryCore`:** if a reload writes it and a request reads it, it belongs in `wiring`, not on the struct. [wiring_test.go](wiring_test.go) is the `-race` harness that says so out loud.

## Let's Encrypt certificates

`CertificateHosts` and `CertificateCache` exist for the same reason. `autocert.Manager` is built **once**, when the HTTPS server starts, and lives for the whole process — but it consults `HostPolicy` on every TLS handshake and `Cache` on every certificate read or write. So anything captured by *value* at construction is frozen forever, while anything reached through a function or an interface follows the configuration.

These used to be `autocert.HostWhitelist(domains...)` and `autocert.DirCache(location)`, both built from a boot-time snapshot. The consequence was a real and confusing bug: a non-local domain added through the setup console could not obtain a TLS certificate until the process was restarted. The domain factory was created, the domain answered over HTTP, and HTTPS failed — which read as "configuration changes need a reboot," but had nothing to do with how the configuration was propagated.

The idiom is the one the digital dome already uses: `dome.New(factory.ClientIP, ...)` passes a method value bound to the factory, so it reads through to the live configuration instead of copying a value out of it. Both types take a `ConfigProvider`, which is all they need and what makes them testable with a stub.

Two things are still frozen at boot, by design or by bug: `startHTTPS` will not bind `:443` with zero non-local domains (binding unconditionally would break every developer on a privileged port), so the *first* such domain still needs a restart; and `autocert` reads the ACME contact email once, when it registers the account (BUG-137).

See the [project README](../README.md) for the big picture, and [config](../config/) for the configuration this factory consumes.
