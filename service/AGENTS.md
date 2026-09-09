# service — Agent Notes

See [README.md](README.md) for what this package is. These are the non-obvious rules.

## Never hand `golang.org/x/oauth2` the default HTTP client

`oauth2.Config.Exchange` and `TokenSource(...).Token()` POST to a **remote-actor-declared** `TokenURL`. With no client in the context they fall back to `http.DefaultClient`, which is UNGUARDED — a blind POST SSRF that reaches internal hosts. Always pass a context carrying `remote.NewHTTPClient(...)` under the `oauth2.HTTPClient` key. Both entry points do this through an `oauthHTTPContext` helper: [import_oauth.go](import_oauth.go) (source-actor endpoints — attacker-controlled) and [domain.go](domain.go) (provider endpoints — admin-configured, defense-in-depth). Pass `activityService.AllowPrivateIPs()` so local/self-federation dev still works.

## Import fetches must be same-origin with the source actor

The user's OAuth Bearer token is scoped to the **source server**, and migration collections legitimately reference third-party hosts (a following/blocked list, a boosted post). So every import dereference is gated with `uri.NotSameOrigin(url, sourceOrigin)`: `consumer/importStartup.go` skips off-origin documents at item creation, `Import.ImportAttachments` skips off-origin attachments, and `consumer/importItems.go` re-checks at the fetch sink as belt-and-suspenders. `doAuthorize` likewise origin-pins the OAuth `AuthURL`/`TokenURL` to the actor.

This is about **credential leakage and content-injection, not private-IP SSRF** — a raw `remote.Get` already blocks private IPs by default (see [remote/AGENTS.md](../../../benpate/remote/AGENTS.md)). The residual risk the same-origin gate closes is the token traveling off-origin and foreign content being saved as the user's record.

## `AllowPrivateIPs` comes from the ActivityStream service

`activityService.AllowPrivateIPs()` is the one predicate for "may this instance talk to private addresses" (true only on a local/private hostname). Thread it into any guarded-client construction here rather than re-deriving it.

## A `UserConnection` is a User's own credential for a third-party service — six rules a second provider will need

`UserConnection` holds one row per `(UserID, Type)` with an `IsActive delta.Bool`, a `Status`, a `Data mapof.String` for non-secret remote IDs, and a `Vault` for secrets. Mailchimp is the only type today; `MerchantAccount` has the same shape and has not migrated. What follows is not visible from the code.

**`connect()` runs BEFORE `collection.Save`, not after.** A credential the remote service refuses must never reach the database, because everything downstream reads a stored connection as a working one. (`connectBluesky` in the Following service runs the other way round; do not copy that here.)

**`IsActive` is a `delta.Bool`, and its CHANGE is the trigger.** Write it through the schema — `SetBool` calls `IsActive.Set` — never by assigning the field, or the change is lost and the connect/disconnect never fires. `Status` is deliberately separate: `IsActive` is what the User asked for, `Status` is whether it works, and folding a rejected credential into the switch would make Emissary look like it turned the connection off.

**`IsReady()` means "switched on AND finished being set up".** It tests `Status == READY`, not `!= RECONNECT`, because a connection whose credential works but whose setup never completed is still `PENDING` (D48) — a real value, assigned by the constructor, never the empty string. Since D47, "set up" includes the webhook on a public domain and deliberately excludes it on a local one (D45) — that decision lives in `mailchimp_setup`'s ordering, not in the predicate, because the model has no host to ask. Every consumer — the inbound webhook, the outbound sync — leans on that distinction.

**There is deliberately no `Fields()` projection, and a test asserts its absence.** `QueryBuilder` projects with `T.Fields()`, and `MerchantAccount`'s projection omits `vault` — which here would make a configured connection read as unconfigured, and compare a presented webhook secret against an empty string.

**The record is HARD deleted, and is not `Importable`.** It holds a live credential, so a soft-deleted copy keeps that credential readable by anything querying without `notDeleted`; hard delete is also what lets `{userId, type}` be a plain unique index instead of a partial one. And an account import that installed someone else's API key would hand this server a credential it should not have.

**A failed remote teardown is reported, never propagated.** Both `Delete` and the pause path do this: a User whose key was already revoked at the far end must still be able to disconnect, and refusing would strand them with a connection they cannot switch off.

Two further notes on the write paths. `Save` is reached *only* from the settings form, so there are no background writes to keep off the network — which is why an active connection is re-proved on every save, and why that re-run is what restores anything the User deleted by hand at the far end. And a queue task that discovers a rejected credential must write `Status` through `collection.Save` directly, **not** through `UserConnection.Save`, which would call `connect()` and reach for the same credential that was just refused.

## A Follower's email address is stored lowercased, and every lookup normalizes to match

`Follower.Save` is the one writer and it applies `model.NormalizeEmailAddress` (trim + lowercase) to `Actor.EmailAddress` — and to `Actor.ProfileURL` **only when `Method == EMAIL`**, because for everyone else `ProfileURL` is a real URL whose path is case-sensitive. `LoadByEmailAddress` normalizes its argument; `LoadByActor` does not (it serves ActivityPub too), so a caller passing an email to it — `handler.PostEmailFollower` through `LoadOrCreate` — normalizes first. Mailchimp hashes members on the lowercased address, so an inbound webhook naming `Sarah@Connor.MIL` must find the row stored as `sarah@connor.mil`; before this rule it silently did not. Rows written before the rule are brought in line by upgrade slot 30.

`Follower.Data["secret"]` — the unsubscribe-link token that `LoadBySecret` checks — is plaintext by design; see [model/AGENTS.md](../model/AGENTS.md).

## `User.GetJSONLD()` output is fingerprinted — keep it deterministic

Every `User.Save` hashes `GetJSONLD()` into `ProfileFingerprint`; a changed hash federates an ActivityPub `Update` to all followers ([user.go](user.go) Save, [user_activitypub.go](user_activitypub.go) `sendProfileUpdate`, spec PROFILE-UPDATE-FEDERATION.md). Adding anything volatile or per-save (timestamps, counters, random values) to `User.GetJSONLD()` makes every save — including signin bookkeeping — spam followers with Updates. `TestUser_CalcProfileFingerprint` pins which fields participate; update it when the actor document gains a field.

## TAG rules only exist in the full-document key set

`model.ActorMatchKeys` deliberately excludes content (TAG) keys — it answers "is this actor filtered?", nothing more. Any enforcement surface that should honor TAG rules (newsfeed ingest, notifications, render labels) must evaluate `model.DocumentMatchKeys` / `Rule.Disposition` on the (unwrapped) payload, or TAG rules silently never fire there. A surface built on `ActorDisposition` alone looks complete and passes every identity-rule test while ignoring hashtag rules entirely.

## A boolean in a Theme's `themeData` needs an explicit `default`, or its first save flips it

A Theme's hjson schema layers `themeData` properties onto the Domain's own wildcard object ([build/builder_admin_domain.go](../build/builder_admin_domain.go) `schema()`), and the settings form is a tab layout — so saving **any** tab posts **every** field. `model.Domain.ThemeData` starts out an empty `mapof.Any`, and an absent boolean reads back as `false`. A toggle meant to start ON therefore renders OFF on a pristine Domain, and the owner's first save of an unrelated tab writes that phantom `false` — silently blanking whatever the flag controls.

Declaring `default:true` on the property is what fixes it, and **both** halves read that one declaration: `form/widget.Toggle` falls back to the schema element's `DefaultValue()` when the object carries no value, and [build/builder_common.go](../build/builder_common.go) `ThemeData` falls back to `theme.Schema.GetElement("themeData." + token).DefaultValue()`. Keep them in step — a default that only one side honors is worse than none, because the page and the form that configures it then disagree. `ThemeData` returns a **string**, so compare against `"true"`, never truthiness.

Note that `schema.Boolean.Default` does *not* surface through `schema.Schema.Get`: `getProperty_Boolean` errors on a missing map key rather than falling through to the default, which is why both fallbacks are written out by hand instead of coming for free.
