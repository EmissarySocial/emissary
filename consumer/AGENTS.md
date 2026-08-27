# consumer — Notes for AI Agents

See [README.md](README.md) for what this package is: the worker side of the [Turbine](https://github.com/benpate/turbine) queue. Each task name in [consumer.go](consumer.go) dispatches through a `With*` wrapper into a handler function; the wrappers are the package's real architecture.

## Every handler must assume it will run again

Turbine retries any task that returns `queue.Error` or `queue.Requeue`, so handlers must be idempotent. Two mechanisms do most of the work: `WithSession` runs the handler inside `factory.WithTransaction`, and returning a `queue.Result` with a non-nil `Error` aborts that transaction — a failed attempt rolls back its own writes, so the retry starts clean. When a handler deliberately commits partial progress and retries anyway (see the `closeTask` helper in [importItems.go](importItems.go), which records the item's outcome then returns `queue.Requeue(0)`), it must make that progress durable state the next run keys off, not in-memory bookkeeping.

## Result semantics: Failure is permanent, Error is retryable

`queue.Failure` means "retrying can never help" (malformed args, invalid ObjectID); `queue.Error` means "try again later"; `queue.Ignored` means "not my task". The [utilities.go](utilities.go) `requeue(err)` helper maps derp error classes for HTTP-backed tasks: 429 → `queue.Requeue(delay)`, other 4xx → `Failure`, everything else → `Error`. Misclassifying a permanent error as retryable leaves a task looping in the queue forever.

## The domain factory comes from the "hostname" task argument

`getHostnameFromArgs` in [utilities.go](utilities.go) resolves the factory for `WithFactory` and everything stacked on it. It reads `"hostname"` first, then the legacy `"host"` name, then falls back to the hostname of an `"actor"` URL; values may be bare hostnames or full URLs and are normalized with `uri.Hostname()`. New producers always pass `"hostname": factory.Hostname()`. The `"host"` fallback exists so tasks persisted under the old argument name still drain — removing it strands any such rows still in the queue, so confirm the stored queue is empty first.

## A missing hostname is `Failure`, an unknown hostname is `Error`

`WithFactory` hard-fails when no hostname argument exists (no retry can supply one) but returns a retryable `Error` when `ByHostname` misses — the domain may simply not be loaded yet (config reload, another node). Keep that asymmetry.

## Tasks enqueued from inside a handler go through `postcommit.Publish`

`WithSession`'s transaction carries the post-commit spool (see the repo-wide rule in [../AGENTS.md](../AGENTS.md)): chained tasks like `PollFollowing-Record` from [pollFollowing-index.go](pollFollowing-index.go) or `ImportItems` from [importStartup.go](importStartup.go) are released to the queue only after the enclosing transaction commits. A direct `queue.NewTask` from inside a handler can run against data the commit has not made visible yet. The `Schedule*` tasks use the queue directly because they run outside any transaction.

## `PublishRealtimeMessage` only works in-process — producers must use `queue.WithInline()`

The task ([publishRealtimeMessage.go](publishRealtimeMessage.go)) delivers to `factory.RealtimeBroker()`, which holds this process's live SSE sockets. A stored, retried, or cross-node run would nudge nobody. Topics travel as the integer constants from [../realtime/constants.go](../realtime/constants.go), so never renumber them.

## New tasks need a case in `PreProcessor` too

[preprocessor.go](preprocessor.go) assigns a priority only when `task.Priority == -1`. Priorities ≤ 32 may run immediately when the queue is idle; anything at 64 or above is always written to storage first. A delivery-style task without a case here gets no tuned priority, and a task whose name is not in the [consumer.go](consumer.go) switch returns `queue.Ignored()` silently — renaming a task name (or its argument names) strands already-queued rows unless a fallback drains them, which is exactly why the `"host"` fallback exists.

## Scheduling is idempotent via task signatures

`scheduler_MakeDailyTasks` / `scheduler_MakeHourlyTasks` in [schedule.go](schedule.go) publish with `queue.WithSignature("DAILY:<date>" / "HOURLY:<hour>")`, so repeated boots and the daily re-priming cannot double-schedule. Any new recurring task should either ride these batches (as the per-domain tasks in [scheduleDaily.go](scheduleDaily.go) do) or carry its own signature. `ScheduleStartup` is an intentionally empty hook, published on every boot — it is where the next one-time migration goes; do not delete it as dead code.

## `RecycleDomain` is a mass purge — treat it with respect

[recycle.go](recycle.go) hard-deletes every record soft-deleted more than 30 days ago across all of `factory.Collections()`, nightly. Most partial indexes filter on `deleteDate: 0`, which excludes exactly the rows this queries, so each collection is a COLLSCAN — that is why `queries.Recycle` runs under a generous timeout and why the loop reports-and-continues instead of failing fast (one chronically slow collection must not starve the rest; keep that structure). On a database with old accumulated soft-deletes, the first run is large and irreversible. Federation side effect: once a soft-deleted Stream is purged, peers get a 404 instead of a Tombstone.

## Notification retention is uniform and the per-user cap lives in the daily task

[purgeNotifications.go](purgeNotifications.go) ages out read and unread notifications alike at 90 days (deliberate — do not "fix" unread being purged) and then trims each User to `notificationCapPerUser` (2000) as a flood backstop, deleting read rows before unread. The cap is enforced here, in the daily task, and NOT in the notification hot path — that placement is a decision (see `emissary-specs/NOTIFICATION-FLOOD-CONTROL.md`), not an oversight. `PurgeOverCap` treats a non-positive cap as disabled so a misconfiguration cannot wipe the collection.

## Import fetches are origin-gated — the rule lives in service/AGENTS.md

[importStartup.go](importStartup.go) skips cross-origin documents at item creation and [importItems.go](importItems.go) re-checks `uri.NotSameOrigin` at the fetch sink so the user's source-scoped OAuth Bearer token can never travel off-origin. The full reasoning (and the service-side gates) is in [../service/AGENTS.md](../service/AGENTS.md) — keep both check sites; the duplication is belt-and-suspenders, not redundancy.

## Outbound delivery filters through the sending actor's rules

`WithSender` in [wrappers.go](wrappers.go) binds the send locator with `BoundToSender(args["actor"])` so recipient resolution respects the sender's block rules, and constructs the hannibal sender with `AllowPrivateIPs` from the server factory — FALSE in production, true only for local/dev federation on a private network. Both `Outbox:SendTo*` task names come from `hannibal/sender` constants; use the constants, not string literals.
