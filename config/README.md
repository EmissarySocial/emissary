# config

This package defines Emissary's server-level configuration — the set of domains served by this instance, plus the storage, database, and command-line settings that the server reads at startup. `Config` is the top-level structure; the accessors and loaders here parse it from its various sources and supply sensible defaults.

## Storage engines

`Storage` has two implementations, chosen by the scheme of the `--config` location. Both do the same three things: read the configuration once at boot, publish it on a channel, and keep publishing whenever it changes underneath the running process.

- **`FileStorage`** (`file://`) reads HJSON from disk and watches it with fsnotify. Single-server only.
- **`MongoStorage`** (`mongodb://`, `mongodb+srv://`) reads one document from a collection and watches it with a MongoDB change stream. This is what keeps a multi-server cluster in step: whichever node writes the configuration, every other node learns about it without restarting.

Both publish through `updateChannel`, which holds a single slot and always keeps the newest entry. Publishing never blocks — see the RULE on `updateChannel.notify` for why a parked watcher goroutine is not merely slow but fatal.

## Rules worth knowing before you edit this package

- **Neither watcher may ever be able to stop.** `MongoStorage.watch` and `FileStorage.watch` are the same supervised loop: reopen on every failure, back off with `watchRetryDelay`, resynchronize with a full read on every reopen, and exit only on `Close()`. Each watcher has a *quiet* death the supervision exists for — the change stream can end with no error at all (`Next()` false, `Err()` nil) when the server closes it, and an fsnotify watch detaches on every atomic-rename save (vim, VS Code, `sed -i`), after which no event ever arrives again. Both used to leave a process running a frozen configuration, with nothing in the log, until someone rebooted it.
- **Change streams need a replica set.** A standalone `mongod` cannot open one, so configuration never propagates between nodes on one. A single-member replica set is fine, and needs `?directConnection=true` in the connect string.
- **`Source` and `Location` are node-local.** They describe where this process found its configuration, are deliberately never persisted, and must be re-stamped onto every value that leaves storage.
- **Copy a `Config` before handing it out.** A plain struct assignment shares the backing storage of every map and slice, so a caller that edits its "copy" reaches into the live configuration of a running server. Use `Config.Copy()`.
- **`Storage.Write` is a compare-and-swap, not a blind replace.** Every save carries the `Revision` it was based on; a save whose revision no longer matches the stored document returns a 409 (`derp.IsConflict`) and changes nothing. Before this, two nodes saving from the same base silently lost one of the two changes — up to and including a domain's `MasterKey`, which lives nowhere else. `Write` returns the configuration *as stored* (revision incremented); callers keep the returned value, or their next save conflicts against their own write. Legacy documents without a `revision` field match the Revision-0 leg once and are stamped on first write. The file engine holds the same contract best-effort (the check and write are not atomic without file locking; file storage is single-server by design).
- **This package never calls `os.Exit`.** Every constructor failure — an unsupported scheme, an unreachable database, a missing file without `--setup`, a corrupt file — comes back as an error carrying the operator-facing guidance, and `main` decides whether it is fatal. That is what makes every failure branch here testable, including the `--setup` bootstrap rule.

## Tests

The MongoDB tests skip when no database is reachable, so `go test ./...` still passes without one. They need a replica set to cover the watcher; run them against `mongodb://localhost:27017/?directConnection=true`.

See [emissary-specs/projects/CONFIG-INTEGRATION-TESTS.md](../../emissary-specs/projects/CONFIG-INTEGRATION-TESTS.md) for the proposed suite covering the rest of the path — whether an applied configuration actually changes what the server does.

See the [project README](../README.md) for the big picture.
