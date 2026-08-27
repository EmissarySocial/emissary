# realtime — Notes for AI Agents

See [README.md](README.md) for what this package is. The `Broker` tracks connected SSE clients and fans realtime messages out to them; the HTTP side lives in [../handler/sse.go](../handler/sse.go) and the routes in [../server.go](../server.go).

## Messages are droppable nudges, never payloads

A `Message` means "something changed, refetch" — the browser responds by re-requesting the page fragment. Delivery in `notifySSE` ([broker.go](broker.go)) is a non-blocking send into each client's 16-slot buffer; a full buffer silently drops the nudge and the next event (or a page load) brings the client current. Do not build anything on these messages that cannot tolerate a drop, and do not make delivery blocking — the non-blocking send is what keeps one wedged client from stalling everyone.

## Clients are keyed by object ID and topic, and `Client.StreamID` is not always a Stream

The broker's `objects` map is keyed by whatever ObjectID the client watches — a Stream's, a User's, or an Import's — despite the field name. `TopicAll` (0) subscribes to every topic for that object; otherwise the client's topic must equal the message's. Topic constants in [constants.go](constants.go) are deliberately non-sequential (`TopicFollowingUpdated` is 8) and travel as integers inside stored `PublishRealtimeMessage` task args, so never renumber an existing one.

## Broker delivery has two live entry paths — new publishers use the post-commit task

`Broker.Send()` delivers synchronously and is the target of the `PublishRealtimeMessage` inline task (see [../consumer/publishRealtimeMessage.go](../consumer/publishRealtimeMessage.go)), which services publish via `postcommit.Publish(..., queue.WithInline())` so the nudge fires only after the transaction commits. The `updateChannel` path (fed by `factory.SSEUpdateChannel()` from several service files and the `queries.Watch*` change streams) still exists but sends mid-transaction — a browser nudged that way can refetch before the commit lands and render stale data. Convert channel sends to the post-commit task when touching them; do not add new ones.

## Never close a client's `WriteChannel` from the broker

`notifySSE` runs on its own goroutine per message and may hold a just-removed client in its delivery snapshot; closing the channel from the `RemoveClient` arm would panic with "send on closed channel". The handler's read loop exits on its request context and the channel is garbage-collected. The `RULE` comment in [broker.go](broker.go) marks this; leave the pattern alone.

## Snapshot under the read lock, send outside it

`notifySSE` copies the matching clients under `mutex.RLock`, releases the lock, then sends. Holding the lock across sends would serialize delivery against every connect/disconnect. Likewise the `TopicNewReplies` 2-second settle delay is a deliberate hack that must stay off the `listen()` goroutine — `listen()` hands each message to `go b.notifySSE(...)` precisely so that delay (and slow clients) cannot block connect/disconnect handling.

## `Message.Event` conventions differ by topic

Stream-ish topics (`Updated`, `ChildUpdated`, `NewReplies`, `FollowingUpdated`, `ImportProgress`) set `Event` to the object's hex ID, and templates subscribe with `hx-trigger="sse:{{.ID.Hex}}"`. The inbox and notification topics leave `Event` empty, producing the default `message` event that templates catch with `hx-trigger="sse:message"`. [message.go](message.go) constructors encode this; changing an event name breaks templates silently.

## The `/@:userId/sse` route is a deliberate compat alias with no ownership check

Authenticated SSE lives at `/@me/sse...`; the extra `/@:userId/sse` route in [../server.go](../server.go) exists because pages rendered from stale templates still name the User in the URL and would otherwise fall through to `/@:userId/:action`, logging an error on every reconnect, forever. The handler (`ServerSentEvent_Me`) never reads the URL param — it streams the signed-in User from the auth cookie — so the alias cannot leak another user's events. Do NOT add a 404-on-mismatch guard: that 404 is itself a logged error and regenerates the exact flood the alias exists to stop (realistic trigger: switching accounts with an old tab open). Remove the alias only once stale pages and third-party skins have aged out.

## Broken SSE clients reconnect forever — two amplifiers to remember

Git-backed template folders are pinned at process start (`Filesystem.Watch` only supports the FILE adapter), so a template fix that changes an SSE URL needs a server restart before production stops serving the old markup. And the vendored htmx SSE extension's reconnect backoff is locally PATCHED (`Math.pow(2, retryCount)` — upstream's `2 ^ retryCount` is XOR and never backs off) in both copies under [../_embed/templates/theme-global/resources/](../_embed/templates/theme-global/resources/) (`htmx/ext/sse.js` and `htmx-1.9.12/ext/sse.js`); re-apply the patch when upgrading htmx.

## The broker is per-process state

Each domain factory owns one `Broker`, and it only knows about sockets attached to this process. Anything that must reach clients on another node cannot go through this package as it stands — which is also why the `PublishRealtimeMessage` task runs inline instead of through queue storage.
