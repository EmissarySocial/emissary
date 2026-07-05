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

## List endpoints must filter visibility inside the query, not per-object.

A list handler can't call `userCanStream` on results it already fetched — by then the leak is the query itself. `Stream.QueryByUser` ([service/stream_mastodon.go](../../service/stream_mastodon.go)) is the choke point for Mastodon list reads: it takes the caller's `Authorization` and ANDs in `visibilityCriteria` (owner/domain-owner see all; everyone else gets `publishDate < now < unpublishDate` plus `defaultAllow ∈ caller's permissions`, failing closed). Any new list query must either go through it or apply the same predicate — filtering on `ownerId` alone is the IDOR that hit `GetAccount_Statuses`.

## `getStreamFromURL` resolves the domain from the *stream URL's* host — known gap.

[utils.go](utils.go) parses the caller-supplied URL and calls `ByHostname(parsedURL.Host)`, so the factory and permission service can belong to a different domain than the request. The per-handler authorization now gates content, so this is not an open confidentiality hole, but two things remain to revisit when returning to this API: a caller can still trigger a lookup against an arbitrary domain on the server, and `GetStatus` evaluates `UserCan` with a foreign domain's permission service against the caller's home-domain `Authorization` (the UserID is meaningless in that domain).

Used by `GetStatus`, `DeleteStatus`, and `PostStatus_Translate`. `GetStatus_Source` and `PutStatus` instead resolve from the request host (`t.Host`) and load via `streamService.LoadByURL` directly.

## Every implemented handler must use one of three authorization shapes

A full cross-handler pass confirmed the package holds to this; keep it that way when implementing stubs:

1. **Caller-scoped queries/loads** — the service call takes `auth.UserID` (or `auth.ClientID`) as a filter: followings, rules (blocks/mutes), folders/lists, news feed, responses, markers, profile edits.
2. **Stream gates** — `userCanStream` for reads, `userOwnsStream` for writes, per the sections above.
3. **Query-level visibility criteria** — list queries like `Stream.QueryByUser`, per the section above.

Creates must pin ownership from the token (`folder.UserID = auth.UserID`, `rule.UserID = auth.UserID`, `following.UserID = auth.UserID`) — never from a transaction field.

## Every Mastodon-created post is world-readable — `PostStatus` ignores `visibility`.

`PostStatus` hard-codes the `outbox-message` template, whose `view` action is `anonymous` in **every** state, and silently discards `txn.PostStatus.Visibility` (`public | unlisted | private | direct`). A client posting a followers-only or direct status gets a public post. Until visibility is mapped to restricted templates/roles, reject non-`public` visibility values instead of silently publishing.

Template `view` audit (2026-07-04): the restriction machinery itself is sound. `calcDefaultAllow` and `UserCan` both read the same state-aware `AccessList` (`roles` + `stateRoles[state]`, anonymous short-circuits), recomputed on every save. `stream-article-base` restricts unpublished articles to `author`/`editor` and opens `viewer` only when `published`; `stream-photograph`/`stream-collection` never grant `anonymous` unless shared. The gap is only that Mastodon posts never use any of it.

## Open follow-ups (not yet done)

- **`PostStatus` visibility mapping** (or explicit rejection), per the section above.
- **`UserCan` has no publish-window check.** By-ID reads (`GetStatus`, source, translate) of an anonymous-`view` stream pass even when the stream is unpublished or expired; only list queries filter `publishDate`/`unpublishDate`. Low exposure today (the create flows publish immediately), but add the window to `userCanStream` for non-owners for defense in depth.
- **`getStreamFromURL` hostname trust**, per the section above.
- **`GetAccount` ignores `user.IsPublic`.** Any token can resolve any local profile URL to public-field data (`User.Toot()` exposes no email). This matches Mastodon semantics, but revisit if Emissary ever promises non-discoverable profiles.
- **`PostStatus` ignores `ScheduledAt` semantics.** It sets `PublishDate` from the field, then publishes immediately anyway. Feature gap, not a security issue.
