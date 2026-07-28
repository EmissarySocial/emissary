# service — Agent Notes

See [README.md](README.md) for what this package is. These are the non-obvious rules.

## Never hand `golang.org/x/oauth2` the default HTTP client

`oauth2.Config.Exchange` and `TokenSource(...).Token()` POST to a **remote-actor-declared** `TokenURL`. With no client in the context they fall back to `http.DefaultClient`, which is UNGUARDED — a blind POST SSRF that reaches internal hosts. Always pass a context carrying `remote.NewHTTPClient(...)` under the `oauth2.HTTPClient` key. Both entry points do this through an `oauthHTTPContext` helper: [import_oauth.go](import_oauth.go) (source-actor endpoints — attacker-controlled) and [domain.go](domain.go) (provider endpoints — admin-configured, defense-in-depth). Pass `activityService.AllowPrivateIPs()` so local/self-federation dev still works.

## Import fetches must be same-origin with the source actor

The user's OAuth Bearer token is scoped to the **source server**, and migration collections legitimately reference third-party hosts (a following/blocked list, a boosted post). So every import dereference is gated with `uri.NotSameOrigin(url, sourceOrigin)`: `consumer/importStartup.go` skips off-origin documents at item creation, `Import.ImportAttachments` skips off-origin attachments, and `consumer/importItems.go` re-checks at the fetch sink as belt-and-suspenders. `doAuthorize` likewise origin-pins the OAuth `AuthURL`/`TokenURL` to the actor.

This is about **credential leakage and content-injection, not private-IP SSRF** — a raw `remote.Get` already blocks private IPs by default (see [remote/AGENTS.md](../../../benpate/remote/AGENTS.md)). The residual risk the same-origin gate closes is the token traveling off-origin and foreign content being saved as the user's record.

## `AllowPrivateIPs` comes from the ActivityStream service

`activityService.AllowPrivateIPs()` is the one predicate for "may this instance talk to private addresses" (true only on a local/private hostname). Thread it into any guarded-client construction here rather than re-deriving it.

## TAG rules only exist in the full-document key set

`model.ActorMatchKeys` deliberately excludes content (TAG) keys — it answers "is this actor filtered?", nothing more. Any enforcement surface that should honor TAG rules (newsfeed ingest, notifications, render labels) must evaluate `model.DocumentMatchKeys` / `Rule.Disposition` on the (unwrapped) payload, or TAG rules silently never fire there. A surface built on `ActorDisposition` alone looks complete and passes every identity-rule test while ignoring hashtag rules entirely.
