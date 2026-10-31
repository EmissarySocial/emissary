# Federation

Last verified against Emissary commit `52238532` on 2026-08-09.

Emissary supports many standard ways of following and sharing content.  The goal is to accept the highest-fidelity connections from as many sites as possible, whatever specific protocols they are using at the time.  In many cases, Emissary will support several competing standards, defaulting to its preferred methods, but accepting connections in whatever form they are offered.

## ActivityPub

The [ActivityPub](https://activitypub.rocks) standard is currently the most common and popular protocol for sharing information between users on various websites.  Emissary currently supports the following Activities defined in [the ActivityPub spec](https://www.w3.org/TR/activitypub).  Other unrecognized activities are ignored.

| Activity | Sending | Receiving |
| -------- | ------- | --------- |
| [Accept](https://www.w3.org/TR/activitypub/#accept-activity-inbox)/Follow | When Emissary receives a follow request, it adds a new "Follower" record and sends a corresponding `Accept` activity to the original server. | When Emissary receives an `Accept` activity tied to a `Follow` activity, it mark the corresponding `Following` record as active. Other forms of `Accept` are ignored.|
| [Add](https://www.w3.org/TR/activitystreams-vocabulary/#dfn-add) | Emissary does not send `Add` activities. | When an account the user follows sends a public `Add`, Emissary loads the added object and backfills the conversation context named in its `target` ([FEP-7888](https://w3id.org/fep/7888)). That context must be a collection attributed to the sender. Inbound `Remove` activities are ignored. |
| [Announce](https://www.w3.org/TR/activitystreams-vocabulary/#dfn-announce) | Emissary sends an `Announce` whenever a person shares a post, addressed to Public and copied to their followers and the post's author. Search actors announce new results to their followers. A Stream actor announces its child Streams as they are published when its Template sets `boost-children`, and rebroadcasts the activities it receives when its Template sets `boost-inbox`. | Emissary handles an inbound `Announce` the same way as a `Like`, recording it in the Stream's `shares` collection. Most `Announce` activities are public, so a share from an account the recipient does not follow is usually not recorded, although the post's author is still notified. |
| [Block](https://www.w3.org/TR/activitystreams-vocabulary/#dfn-block) | Emissary does not currently send `Block` activities. | Emissary does not currently process inbound `Block` activities on user inboxes. |
| [Create](https://www.w3.org/TR/activitypub/#create-activity-inbox)/* | Emissary's publisher service sends `Create` activities to all followers whenever a new Stream is created.  The object type is determined by the Stream's Template. | When Emissary receives a `Create` that is not public, it stores it as a direct message. A public post is added to the user's newsfeed only if the user follows the sender. A reply to a post on this server joins that post's replies and conversation context, unless the post has closed replies, in which case the reply is dropped and the post's author is not notified. If the activity mentions (tags) the recipient, or replies to one of the recipient's own posts, Emissary also creates a Notification for the recipient, regardless of whether they follow the sender. |
| [Delete](https://www.w3.org/TR/activitypub/#delete-activity-outbox)/* | Emissary's publisher service sends a `Delete` activity, with a `Tombstone` as its object, to all followers whenever a Stream is unpublished or a published Stream is deleted. | When Emissary receives a public `Delete`, it re-fetches the object and removes its cached copy if the object is gone. The object must be on the sender's own origin. When an actor deletes itself, Emissary removes that actor from its cache. |
| [Dislike](https://www.w3.org/TR/activitystreams-vocabulary/#dfn-dislike) | Emissary sends a `Dislike` activity to a remote Inbox whenever a person responds NEGATIVELY to an external post. It is delivered only to the post's author, with no `to` or `cc`. | User inboxes do not record inbound `Dislike` activities, but the post's author is still notified. Configured Stream actors can receive and rebroadcast them. |
| [EmojiReact](https://w3id.org/fep/c0e0) | Emissary does not send `EmojiReact` activities. | Emissary treats an inbound `EmojiReact` or `EmojiReaction` as a `Like`, and discards the emoji. |
| [Follow](https://www.w3.org/TR/activitypub/#follow-activity-outbox) | Emissary sends a `Follow` activity to a remote Inbox whenever a person requests to follow another ActivityPub Actor. | When Emissary receives a `Follow` activity, it validates the request, creates a new `Follower` record in the user's inbox, sends a corresponding `Accept` message to the originating server, and creates a Notification for the followed user.  Emissary does not currently allow user's to manually approve/disapprove follow requests. |
| [Like](https://www.w3.org/TR/activitystreams-vocabulary/#dfn-like) | Emissary sends a `Like` activity to a remote Inbox whenever a person responds POSITIVELY to an external post. It is delivered only to the post's author, with no `to` or `cc`. | Emissary records an inbound `Like` in the liked Stream's `likes` collection when the `Like` is not addressed to Public, or comes from an account the recipient follows. A public `Like` from anyone else is accepted but not recorded. If the recipient follows the sender, the liked post is also added to their newsfeed. If the liked object is one of the recipient's own posts, Emissary also creates a Notification for the recipient. |
| [Move](https://www.w3.org/TR/activitystreams-vocabulary/#dfn-move) | When a user finishes moving their account to another server, Emissary sends a `Move` to the new account and to each of the user's followers. | When Emissary receives a `Move`, it re-points every local `Following` of the old account to the new one, but only if the new account lists the old one in `alsoKnownAs`. A `Move` into an account on this server completes an import that account has in progress. |
| [Reject](https://www.w3.org/TR/activitystreams-vocabulary/#dfn-reject)/Follow | Emissary sends a `Reject` activity when a user removes a follower through the Mastodon API (`POST /api/v1/accounts/:id/remove_from_followers`). | Emissary does not currently process inbound follow rejections; a declined follow remains pending. |
| [Undo](https://www.w3.org/TR/activitypub/#undo-activity-outbox)/Announce | Emissary sends an `Undo` activity, to the same recipients as the original, whenever a user stops sharing a post. | When Emissary receives an `Undo` activity linked to an `Announce`, it removes the share from the Stream's `shares` collection. Only the original sender can undo it, and the `Announce` must be on the sender's origin. A `Delete` of the `Announce` works the same way. |
| [Undo](https://www.w3.org/TR/activitypub/#undo-activity-outbox)/Block | Emissary does not currently send `Undo` activities linked to `Block`. | Emissary does not currently process inbound `Undo` activities linked to `Block` on user inboxes. |
| [Undo](https://www.w3.org/TR/activitypub/#undo-activity-outbox)/Dislike | Emissary sends an `Undo` activity whenever a user deletes a NEGATIVE `Response` record in their profile. | When Emissary receives an `Undo` activity linked to a `Dislike`, it removes the dislike from the Stream's `dislikes` collection. Only the original sender can undo it, and the `Dislike` must be on the sender's origin. A `Delete` of the `Dislike` works the same way. |
| [Undo](https://www.w3.org/TR/activitypub/#undo-activity-outbox)/Follow | Emissary sends an `Undo` activity whenever a user deletes a `Following` record in their profile. | When Emissary receives an `Undo` activity linked to a follow request, it deletes the corresponding `Follower` record from that user's profile. |
| [Undo](https://www.w3.org/TR/activitypub/#undo-activity-outbox)/Like | Emissary sends an `Undo` activity whenever a user deletes a POSITIVE `Response` record in their profile. | When Emissary receives an `Undo` activity linked to a `Like`, it removes the like from the Stream's `likes` collection. Only the original sender can undo it, and the `Like` must be on the sender's origin. A `Delete` of the `Like` works the same way. |
| [Update](https://www.w3.org/TR/activitypub/#update-activity-outbox)/* | Emissary's publisher service sends an `Update` activity whenever a currently-published Stream is published again. | When Emissary receives an `Update` activity, it updates the corresponding message in that user's Inbox. |
| [Update](https://www.w3.org/TR/activitypub/#update-activity-outbox)/Actor | Emissary sends an `Update` of the actor to a user's followers whenever their profile changes. It is addressed to Public only when the profile is public. | When Emissary receives an `Update` of a `Person`, `Application`, `Group`, `Organization`, or `Service`, it re-fetches that actor from its origin rather than trusting the embedded copy. An actor can only update itself. An `Update` of any other actor type is handled like an `Update` of a post. |

### Delivering to Emissary

Inboxes apply these rules before any row in the table above:

| Rule | Applies to | Details |
| ---- | ---------- | ------- |
| Sign every POST | Every inbox | Use an [HTTP Signature](#http-signatures) made by a key that belongs to the activity's `actor`. Emissary does not accept activities forwarded by another server. |
| Same-origin `id` | Every inbox | The activity's `id` must share its actor's origin. |
| Refusals return `401` | Every inbox | Any refusal at this stage gets `401 Unauthorized`: a missing or bad signature, an `id` on another origin, or a sender the recipient does not accept. The response does not say which. |
| Unsigned `Delete` | Every inbox | Accepted when fetching its object returns `404` or `410`. |
| `emissary:` properties | Every inbox | Removed from every inbound activity. |
| Repeated `id` | User inboxes | Answered with success and not processed again. |
| Farewell `Delete` | User inboxes | A deleted account's `Delete`, whose key can no longer be fetched, is answered `202 Accepted` and otherwise ignored. |

### HTTP Signatures

Emissary signs and verifies requests with [draft-cavage-http-signatures-12](https://datatracker.ietf.org/doc/html/draft-cavage-http-signatures-12), following the SWICG report [ActivityPub and HTTP Signatures](https://swicg.github.io/activitypub-http-signature/). It does not support [RFC 9421](https://www.rfc-editor.org/rfc/rfc9421.html).

| Request | Headers signed | Details |
| ------- | -------------- | ------- |
| Inbound POST | `(request-target)`, `host`, `date`, `digest` | Required. The `Digest` may use SHA-256 or SHA-512, and the `Date` must be less than 12 hours old. |
| Inbound GET | None | Not required. Emissary has no authorized-fetch mode. |
| Outbound POST | `(request-target)`, `host`, `date`, `digest` | Signed with `rsa-sha256` by the sending actor's 2048-bit RSA key, with a SHA-256 `Digest`. |
| Outbound GET | `(request-target)`, `host`, `date` | Signed with `rsa-sha256` by the actor the fetch is made for. Background fetches, such as key lookups, thread backfill, and the anonymous `/.proxy`, are signed by the Application actor. |


## Notifications

Modeled on Mastodon, Emissary maintains a recipient-centric **Notification** for each inbound event that involves a local user: being **mentioned** (tagged) in a post, being **replied to**, having one's own content **liked / disliked / announced (boosted)**, and being **followed**.  Notifications are created regardless of whether the recipient follows the sender, and are filtered by the recipient's block/mute Rules.  An inbound `Undo` or `Delete` retracts the corresponding notification.

Notifications are delivered in real time to open browser tabs via **Server-Sent Events**, and to closed tabs via **Web Push** ([RFC 8291](https://datatracker.ietf.org/doc/html/rfc8291)), using a per-domain VAPID keypair.  They are also exposed through the Mastodon `/api/v1/notifications` endpoints (Dislike has no Mastodon equivalent and is omitted from that API).


## WebFinger

Emissary supports but does not require [WebFinger protocol](https://webfinger.net).  Every Emissary instance includes a **WebFinger server** that provides the publicly-available metadata about the people on that server, and is a **WebFinger client** that can use WebFinger to look up metadata from remote servers.

WebFinger also discovers Stream, Domain, Search, and Application actors, and advertises [FEP-3b86 Activity Intents](https://w3id.org/fep/3b86) where available.

## Additional Federation Features

| Feature | Details |
| ------- | ------- |
| [ActivityPub](https://www.w3.org/TR/activitypub) actors | Users, Streams, Domains, saved Searches, and the Application actor publish profiles, inboxes, outboxes, and followable collections. |
| [MLS over ActivityPub](https://swicg.github.io/activitypub-e2ee/mls) | Users publish KeyPackages and receive MLS direct messages through dedicated collections and SSE topics. |

## ActivityPub Extensions

Emissary publishes and reads these properties from outside the core ActivityStreams vocabulary:

| Extension | Details |
| --------- | ------- |
| Mastodon's `toot:` terms | Actors publish `discoverable`, which is always true, and `indexable`, which follows each user's search-indexing setting. Profile links are published as Mastodon-style `PropertyValue` attachments. |
| GoToSocial's `interactionPolicy` | Each Stream publishes `canReply.automaticApproval`: Public, or only its author when replies are closed. Emissary never publishes `manualApproval`, because it does not answer a `ReplyRequest`. It also reads other servers' policies, and hides its reply button where a policy refuses the user. |
| Quote posts | Emissary reads `quoteUrl`, `quoteUri`, `_misskey_quote`, and [FEP-e232](https://w3id.org/fep/e232) `Link` tags whose `rel` names a quote, but only to hide posts that quote an account the user has blocked or muted. |
| The `emissary:` namespace | Some Emissary documents, including actors, carry properties prefixed `emissary:`. They are for Emissary's own use, and the prefix is not yet declared in `@context`. Other servers should ignore them. |

## Other Ways to Follow and Embed

| Method | Details |
| ------ | ------- |
| Bluesky | A Bluesky handle such as `@alice.bsky.social` is followed through [Bridgy Fed](https://fed.brid.gy), at `bsky.brid.gy`. |
| Hashtags | A `#hashtag` is followed through [tags.pub](https://tags.pub). |
| Email | Visitors can follow by email, confirming through a link that Emissary sends them, and then receive new posts by email. The form posts to `/.follower/new`. |
| oEmbed | Emissary serves [oEmbed](https://oembed.com) in JSON and XML at `/.oembed?url=<permalink>`. |

## RSS and Extensions

Emissary publishes feeds in [RSS 2.0](https://en.wikipedia.org/wiki/RSS), [Atom](https://en.wikipedia.org/wiki/Atom_(web_standard)), and [JSONFeed](https://www.jsonfeed.org) formats. It does not currently follow RSS, Atom, or JSON Feed sources.

**Creating Feeds:** The `view-feed` step adds a feed to any Stream, listing its children. The format is negotiated from the `Accept` header, preferring JSONFeed, then Atom, then RSS. A `format` query parameter of `json`, `atom`, or `rss` overrides it.

## MicroFormats

Emissary's default templates all include standard [MicroFormats](https://indieweb.org/microformats) for all available data points.

## Mastodon API

Emisary implements a subset of the [Mastodon API](https://docs.joinmastodon.org/api/), allowing third-party Mastodon clients to interact with Emissary for all features commonly supported by both Emissary and Mastodon.

## FEPs

[Fediverse Enhancement Proposals](https://w3id.org/fep) extend ActivityPub and ActivityStreams.

| FEP | Status | Notes |
| --- | ------ | ----- |
| [FEP-3b86: Activity Intents](https://w3id.org/fep/3b86) | Implemented | WebFinger for a public profile advertises `Announce`, `Create`, `Dislike`, `Follow`, and `Like` intents. Emissary also uses other servers' intents, so a visitor can respond from their own account. |
| [FEP-d556: Server-Level Actor Discovery Using WebFinger](https://w3id.org/fep/d556) | Implemented | WebFinger exposes the server-level actor. |
| [FEP-2677: Identifying the Application Actor](https://w3id.org/fep/2677) | Implemented | The Application actor is published and discoverable. |
| [FEP-f1d5: NodeInfo in Fediverse Software](https://w3id.org/fep/f1d5) | Implemented | NodeInfo 2.0 is published at `/.well-known/nodeinfo/2.0`. |
| [FEP-0151: NodeInfo in Fediverse Software (2025 edition)](https://w3id.org/fep/0151) | Implemented | NodeInfo 2.1 is published at `/.well-known/nodeinfo/2.1`, and both versions are linked from `/.well-known/nodeinfo`. |
| [FEP-7888: Demystifying the `context` Property](https://w3id.org/fep/7888) | Implemented | Public posts carry a `context`, and inbound `Add` activities backfill it. |
| [FEP-2931: Representing Context with a Collection](https://w3id.org/fep/2931) | Implemented | Each context is a collection, served at `/@<user>/pub/collections/<id>`. |
| [FEP-f228: Backfilling Conversations](https://w3id.org/fep/f228) | Implemented | Emissary crawls a thread's context and reply tree to fill in the conversation. |
| [FEP-5feb: Search Indexing Consent for Actors](https://w3id.org/fep/5feb) | Implemented | Actors publish `indexable` from each user's setting. |
| [FEP-67ff: FEDERATION.md](https://w3id.org/fep/67ff) | Implemented | This document. |
| [FEP-1b12: Group Federation](https://w3id.org/fep/1b12) | Partial | Stream rebroadcasting is available to templates as Content Actors. |
| [FEP-e232: Object Links](https://w3id.org/fep/e232) | Partial | Quote links are read only to hide posts that quote a blocked or muted account. |
| [FEP-7628: Move Actor](https://w3id.org/fep/7628) | Partial | `Move` is sent and received, but actors publish no `movedTo` or `alsoKnownAs`. |
| [FEP-11dd: Context Ownership and Inheritance](https://w3id.org/fep/11dd) | Partial | Contexts carry `attributedTo`. A reply takes the first `context` found within five ancestors rather than the root's, and starts a new context when it finds none. |
| [FEP-b2b8: Long-form Text](https://w3id.org/fep/b2b8) | Partial | Templates publish long-form posts as `Article`. `summary` is omitted, because Mastodon shows it as a content warning, and no `preview` is sent. |
| [FEP-eb48: Hashtags](https://w3id.org/fep/eb48) | Partial | Hashtags are published as `Hashtag` tags. The parser also accepts non-ASCII letters, which the FEP excludes. |
| [FEP-9f9f: Collections](https://w3id.org/fep/9f9f) | Partial | Collections are paged with `first` and `next`, but carry no `last` or `prev`. |
| [FEP-a974: All Actor Types Should Be Followable](https://w3id.org/fep/a974) | Partial | Actors of any type can follow and be followed, but `Update` refreshes the profiles of only five actor types. |
| [FEP-fe34: Origin-Based Security Model](https://w3id.org/fep/fe34) | Partial | An activity's `id` must share its actor's origin, and a `Delete` must name an object on that origin. |
| [FEP-c0e0: Emoji Reactions](https://w3id.org/fep/c0e0) | Not implemented | `EmojiReact` is received as a `Like`, and the emoji is discarded. |
