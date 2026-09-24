# consumer — Notes for AI Agents

See [README.md](README.md) for what this package is: the worker side of the [Turbine](https://github.com/benpate/turbine) queue. Each task name in [consumer.go](consumer.go) dispatches through a `With*` wrapper into a handler function; the wrappers are the package's real architecture.

## Every handler must assume it will run again

Turbine retries any task that returns `queue.Error` or `queue.Requeue`, so handlers must be idempotent. Two mechanisms do most of the work: `WithSession` runs the handler inside `factory.WithTransaction`, and returning a `queue.Result` with a non-nil `Error` aborts that transaction — a failed attempt rolls back its own writes, so the retry starts clean. When a handler deliberately commits partial progress and retries anyway (see the `closeTask` helper in [importItems.go](importItems.go), which records the item's outcome then returns `queue.Requeue(0)`), it must make that progress durable state the next run keys off, not in-memory bookkeeping.

## Result semantics: Failure is permanent, Error is retryable

`queue.Failure` means "retrying can never help" (malformed args, invalid ObjectID); `queue.Error` means "try again later"; `queue.Ignored` means "not my task". The [utilities.go](utilities.go) `requeue(err)` helper maps derp error classes for HTTP-backed tasks: 429 → `queue.Requeue(delay)`, other 4xx → `Failure`, everything else → `Error`. Misclassifying a permanent error as retryable leaves a task looping in the queue forever.

## A permanent failure ends the task SUCCESSFULLY, because `Failure` still reports

`queue.Error` and `queue.Failure` both call `derp.Report` inside turbine's worker, so classifying a permanent error as `Failure` stops the *retry* but not the *reporting* — the same defect is re-filed on every cycle, which is how one signature became the largest single error class on the server (BUG-148). A failure that is understood and permanent must return `queue.Success()` after a `log.Debug`, as the document loop in [pollFollowing-record.go](pollFollowing-record.go) does with `continue`. Use `requeue(err)` only where reporting is still wanted.

`actorError` in [pollFollowing-record.go](pollFollowing-record.go) is the worked example: 429 → `requeue` (the host is throttling, not this record), anything else → record it on the Following and return Success. The 429 check must come **first**, because `derp.IsClientError` is `400 <= code < 500` and therefore covers 429 too, and the service would record it as the record's own failure.

**`derp.Wrap(nil, …)` is NOT nil, and `derp.IsNil` does not catch it.** It builds an `Error` whose `Code` is `ErrorCode(nil)`, which is `0`, carrying no details and no wrapped value — so a `derp.Report` reached on a success path files a record with nothing in it to identify. `CrawlContext` did that 301 times in seven days (BUG-150) because it tested `context.IsCollection()` before it tested `err`. Settle `err != nil` first, and use `derp.WrapIF` anywhere the error may legitimately be nil. Status code `0` in the error log means this and nothing else.

**A 429's `Retry-After` survives `derp.Wrap` and nothing else, in the `derp` this repo pins.** In v0.43.0 `derp.RetryAfter` is a bare type assertion, and `derp.Error.GetRetryAfter` propagates only by delegating to its own `WrappedValue` — so one `fmt.Errorf("%w")` anywhere in the chain zeroes the duration. `derp.ErrorCode` walks the chain with `errors.As` and is unaffected, which is what hides it: `derp.IsTooManyRequests` still answers TRUE on the surviving code and substitutes its one-hour default, so a host asking for 30 seconds is deferred an hour and nothing logs the difference. Until `go.mod` moves past v0.43.0, the repo-wide ban on `fmt.Errorf` is the only thing holding the chain together. Fixed upstream on 2026-09-19 by giving `RetryAfter` the same `errors.As` walk; **delete this note when the dependency is bumped.**

**A remote answering HTML to an ActivityPub request is a 500, not a 4xx.** `remote.Transaction.decodeResponseBody` wraps that case with `derp.WithInternalError()`, and `derp.Wrap` applies its options *after* computing the inner code, so the option wins. This is the largest single error class `CrawlContext` produced, and `derp.IsClientError` matches none of it. The dead-domain rule below fails the same way, from the opposite direction.

## Polling has two clocks, and they own different questions

`Following.NextPoll` answers **"how often do we check this source?"** — a cadence, in hours. Turbine's task retry answers **"did that one HTTP call blip?"** — recovery, in minutes. Keeping them separate is deliberate; the two used to be conflated in an exponential backoff written onto `NextPoll`, at a 1m–256m timescale that matched neither the four-hour sweep nor the 24-hour `PollDuration` (BUG-148).

So `PollFollowing_Record` ends **every** completed attempt by writing the cadence: `SetStatusPollSuccess` on success, and `SetStatusPollError` on any failure. `SetStatusPollError` belongs to the Following service, and it owns the whole choice of what a failure means for the record: `GONE` on a `410`, `FAILURE` otherwise, escalating to `PAUSED` in time, plus the sentence shown to the owner. The consumer never picks a status or writes a message itself. Success and failure both set `NextPoll` one `PollDuration` ahead — a failure uses the *same* cadence as a success, on purpose. Only a `429` hands anything back to turbine, via `requeue`, because a rate limit is the host's throttle and not this record's fault. `FAILURE` exists to tell the owner the feed is broken; it never changes how often we check.

**A dead domain never answers 4xx.** It answers DNS failure, connection refused, TLS handshake failure, or timeout, and `derp` reports all of those as 500. So `derp.IsClientError` is the wrong gate for "did this attempt complete?" — an earlier cut classified only client errors and left the dead-domain case untouched, churning an error-log entry every sweep forever. `actorError` now records **every** completed attempt and reserves `derp.Report` for what nothing can account for, which excludes a 4xx and a 2xx that held no Actor.

**Three problem states, and they are not interchangeable** (see `FOLLOWING-STATUS-LIFECYCLE.md` in emissary-specs). `FAILURE` is any failed poll, at the normal cadence. `PAUSED` is 30 days without a successful retrieval **plus** 5 consecutive failures, via `SetStatusPollFailure` → `isUnresponsive`; it backs off to a 90-day `NextPoll` and is deliberately *not* excluded from `pollableCriteria`, because that quarterly retry is the only automatic way back — BUG-148's Defect B was a run of 401s caused by *this server's* own bug, and an abandonment that could not undo itself would have made those follows unrecoverable. `GONE` is an explicit `410` only, via `SetStatusGone`; it *is* excluded from polling, by status and never by a far-future date, and nothing but a `410` earns it — a `401`/`403` is a refusal, not evidence. `LastPolled` is what measures "how long has this been broken", so **it must never be stamped on a failure**: its meaning is "last RETRIEVED", and stamping it on failure silently makes every record look freshly healthy. `BLOCKED` is the user's own block rule (R8), is also excluded from polling, and is the *only* status `Following.Save` refuses outright.

Two more rules follow from this, and a refactor can break either one without a test failing:

**A handler cannot advance `NextPoll` and return `queue.Error` in the same run.** `WithSession`'s transaction is aborted by any result carrying a non-nil `Error`, so the write is rolled back. That is *why* the permanent-failure branch returns `queue.Success()` — the mark, not the return value, is what stops the loop, and a `Failure` would discard the mark and report the error anyway.

**The sweep enqueue must keep its `queue.WithSignature`.** Turbine's retry chain runs up to ~4h15m (`RetryMax` 8, `2^n` minutes) which **outlasts the four-hour sweep**, so without the signature a sweep queues a second task for a Following whose first one is still retrying. The signature is keyed on the FollowingID and nothing coarser — see `pollFollowingSignature`. Storage frees it on completion (`onTaskSucceeded` and `onTaskFailure` both `DeleteTask`), so it can never permanently block a record.

The escalating backoff still exists, and still belongs to `SetStatusFailure` — but only on the **connect** path, where someone has clicked "follow" moments ago and is watching the badge. Do not reuse it for polling.

## The reply-tree crawlers must stay bounded — every safeguard is important

The reply graph is remote-controlled data, so the crawl tasks in [crawlContext.go](crawlContext.go), [crawlUpReplyTree.go](crawlUpReplyTree.go), and [crawlDownReplyTree.go](crawlDownReplyTree.go) carry four defenses that all look removable and are not. (1) A `"depth"` argument capped at `maxCrawlDepth` — the only guard that makes cycles (self-replies, mutual replies, an ancestor listed inside a `replies` collection) mathematically unable to breed tasks forever; without it the production queue once accumulated millions of `CrawlDownReplyTree` rows. (2) The `ascache.FromCache` guard that stops a down-crawl at an already-seen document. (3) The `"force"` argument on the ONE seed task `CrawlUpReplyTree` enqueues — the up-crawl's own `Load` cached that URL moments before, so without the exemption every crawl dies at its seed and the guard silently disables the feature (which is why it was once commented out). (4) `queue.WithSignature` on every crawl enqueue, so concurrent crawls of the same thread collapse instead of multiplying. hannibal's `RangePages` additionally caps pages per collection, because a cycle of NON-empty `next` links defeats its empty-page check from inside a single task.

## The domain factory comes from the "hostname" task argument

`getHostnameFromArgs` in [utilities.go](utilities.go) resolves the factory for `WithFactory` and everything stacked on it. It reads `"hostname"` first, then the legacy `"host"` name, then falls back to the hostname of an `"actor"` URL; values may be bare hostnames or full URLs and are normalized with `uri.Hostname()`. New producers always pass `"hostname": factory.Hostname()`. The `"host"` fallback exists so tasks persisted under the old argument name still drain — removing it strands any such rows still in the queue, so confirm the stored queue is empty first.

## A missing hostname is `Failure`, an unknown hostname is `Error`

`WithFactory` hard-fails when no hostname argument exists (no retry can supply one) but returns a retryable `Error` when `ByHostname` misses — the domain may not be loaded yet (config reload, another node). Keep that asymmetry.

## Tasks enqueued from inside a handler go through `postcommit.Publish`

`WithSession`'s transaction carries the post-commit spool (see the repo-wide rule in [../AGENTS.md](../AGENTS.md)): chained tasks like `PollFollowing-Record` from [pollFollowing-index.go](pollFollowing-index.go) or `ImportItems` from [importStartup.go](importStartup.go) are released to the queue only after the enclosing transaction commits. A direct `queue.NewTask` from inside a handler can run against data the commit has not made visible yet. The `Schedule*` tasks use the queue directly because they run outside any transaction.

## `PublishRealtimeMessage` only works in-process — producers must use `queue.WithInline()`

The task ([publishRealtimeMessage.go](publishRealtimeMessage.go)) delivers to `factory.RealtimeBroker()`, which holds this process's live SSE sockets. A stored, retried, or cross-node run would nudge nobody. Topics travel as the integer constants from [../realtime/constants.go](../realtime/constants.go), so never renumber them.

## New tasks need a case in `PreProcessor` too

[preprocessor.go](preprocessor.go) assigns a priority only when `task.Priority == -1`. Priorities ≤ 32 may run immediately when the queue is idle; anything at 64 or above is always written to storage first. A delivery-style task without a case here gets no tuned priority, and a task whose name is not in the [consumer.go](consumer.go) switch returns `queue.Ignored()` silently — renaming a task name (or its argument names) strands already-queued rows unless a fallback drains them, which is why the `"host"` fallback exists.

## Scheduling is idempotent via task signatures

`scheduler_MakeDailyTasks` / `scheduler_MakeHourlyTasks` in [schedule.go](schedule.go) publish with `queue.WithSignature("DAILY:<date>" / "HOURLY:<hour>")`, so repeated boots and the daily re-priming cannot double-schedule. Any new recurring task should either ride these batches (as the per-domain tasks in [scheduleDaily.go](scheduleDaily.go) do) or carry its own signature. `ScheduleStartup` is an intentionally empty hook, published on every boot — it is where the next one-time migration goes; do not delete it as dead code.

## `RecycleDomain` is a mass purge — treat it with respect

[recycle.go](recycle.go) hard-deletes every record soft-deleted more than 30 days ago across all of `factory.Collections()`, nightly. Most partial indexes filter on `deleteDate: 0`, which excludes the rows this queries, so each collection is a COLLSCAN — that is why `queries.Recycle` runs under a generous timeout and why the loop reports-and-continues instead of failing fast (one chronically slow collection must not starve the rest; keep that structure). On a database with old accumulated soft-deletes, the first run is large and irreversible. Federation side effect: once a soft-deleted Stream is purged, peers get a 404 instead of a Tombstone.

## Notification retention is uniform and the per-user cap lives in the daily task

[purgeNotifications.go](purgeNotifications.go) ages out read and unread notifications alike at 90 days (deliberate — do not "fix" unread being purged) and then trims each User to `notificationCapPerUser` (2000) as a flood backstop, deleting read rows before unread. The cap is enforced here, in the daily task, and NOT in the notification hot path — that placement is a decision (see `emissary-specs/NOTIFICATION-FLOOD-CONTROL.md`), not an oversight. `PurgeOverCap` treats a non-positive cap as disabled so a misconfiguration cannot wipe the collection.

## Import fetches are origin-gated — the rule lives in service/AGENTS.md

[importStartup.go](importStartup.go) skips cross-origin documents at item creation and [importItems.go](importItems.go) re-checks `uri.NotSameOrigin` at the fetch sink so the user's source-scoped OAuth Bearer token can never travel off-origin. The full reasoning (and the service-side gates) is in [../service/AGENTS.md](../service/AGENTS.md) — keep both check sites; the duplication is belt-and-suspenders, not redundancy.

## Outbound delivery filters through the sending actor's rules

`WithSender` in [wrappers.go](wrappers.go) binds the send locator with `BoundToSender(args["actor"])` so recipient resolution respects the sender's block rules, and constructs the hannibal sender with `AllowPrivateIPs` from the server factory — FALSE in production, true only for local/dev federation on a private network. Both `Outbox:SendTo*` task names come from `hannibal/sender` constants; use the constants, not string literals.
