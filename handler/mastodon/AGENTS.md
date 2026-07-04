# Mastodon API handlers

Emissary's implementation of the Mastodon-compatible client API, wired into `echo` through `benpate/toot` (the API surface) and `benpate/toot-echo` (route registration and Bearer-scope enforcement). Each handler is a closure over `*server.Factory`.

## Authorization is per-handler, not automatic

**`toot-echo` only enforces the OAuth *scope* (e.g. `read:statuses`); it does NOT check whether the caller may touch the specific object.** Any valid token clears the scope gate, so every handler that loads a user-owned object must authorize the object itself. Skipping this is an IDOR — it was the original bug in `GetStatus_Source` and `PostStatus_Translate`. When adding a handler that loads a stream or message by ID or URL, add the authorization check before returning or mutating anything.

## Reads use `userCanStream`; writes use `userOwnsStream`. Do not mix them.

The two gates answer different questions and are deliberately separate ([utils.go](utils.go)):

- **Reads** (`view`, source, translate) → `userCanStream(..., "view")`, which runs the template's `UserCan` visibility policy. A published post is world-readable; a restricted one follows its template/state access list. Author-only would wrongly hide public posts.
- **Writes** (edit, delete) → `userOwnsStream`, which is author-only (`stream.IsAuthor`) plus a `DomainOwner` escape hatch for moderation. This matches the Mastodon model (a status belongs to one account) and, critically, **ignores the template's access list** — routing writes through `UserCan("edit"/"delete")` could grant a non-author write access if a template shares that role with a group.

## Never use `stream.IsMyself` as an author check.

**`model.Stream.IsMyself` ALWAYS returns `false`** — it means "is this object the user's own profile," which a Stream never is. Using it as an "am I the author?" gate rejects the real author too (this silently broke `PutStatus` and `DeleteStatus`). The author predicate is `stream.IsAuthor(userID)`, which also guards against a zero `authorID` matching a zero-author stream. See [status_authorization_test.go](status_authorization_test.go).

## `getStreamFromURL` resolves the domain from the *stream URL's* host — known gap.

[utils.go](utils.go) parses the caller-supplied URL and calls `ByHostname(parsedURL.Host)`, so the factory and permission service can belong to a different domain than the request. The per-handler authorization now gates content, so this is not an open confidentiality hole, but two things remain to revisit when returning to this API: a caller can still trigger a lookup against an arbitrary domain on the server, and `GetStatus` evaluates `UserCan` with a foreign domain's permission service against the caller's home-domain `Authorization` (the UserID is meaningless in that domain).

Used by `GetStatus`, `DeleteStatus`, and `PostStatus_Translate`. `GetStatus_Source` and `PutStatus` instead resolve from the request host (`t.Host`) and load via `streamService.LoadByURL` directly.

## Open follow-ups (not yet done)

- **Broad IDOR sweep.** Only the three reported status endpoints were audited. The other status, favourite, and mute handlers scope loads by `auth.UserID` (`LoadByURL(session, auth.UserID, ...)`) and look safe, but no deliberate cross-handler pass was made.
- **Template `view`-visibility audit.** The default `stream-outbox-message` template gates `view` to `anonymous`, so read protection for non-public posts depends entirely on those posts using a template or state that restricts `view`. Confirm sensitive-content templates actually do.
- **`getStreamFromURL` hostname trust**, per the section above.
