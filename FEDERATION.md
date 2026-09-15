# Federation

Last verified against Emissary commit `52238532` on 2026-08-09.

Emissary supports many standard ways of following and sharing content.  The goal is to accept the highest-fidelity connections from as many sites as possible, whatever specific protocols they are using at the time.  In many cases, Emissary will support several competing standards, defaulting to its preferred methods, but accepting connections in whatever form they are offered.

## ActivityPub

The [ActivityPub](https://activitypub.rocks) standard is currently the most common and popular protocol for sharing information between users on various websites.  Emissary currently supports the following Activities defined in [the ActivityPub spec](https://www.w3.org/TR/activitypub).  Other unrecognized activities are ignored.

| Activity | Sending | Receiving |
| -------- | ------- | --------- |
| [Accept](https://www.w3.org/TR/activitypub/#accept-activity-inbox)/Follow | When Emissary receives a follow request, it adds a new "Follower" record and sends a corresponding `Accept` activity to the original server. | When Emissary receives an `Accept` activity tied to a `Follow` activity, it mark the corresponding `Following` record as active. Other forms of `Accept` are ignored.|
| [Block](https://www.w3.org/TR/activitystreams-vocabulary/#dfn-block) | Emissary sends a `Block` activity to all followers whenever a user creates a Block in their profile that is shared publicly. | Emissary does not currently process inbound `Block` activities on user inboxes. |
| [Create](https://www.w3.org/TR/activitypub/#create-activity-inbox)/* | Emissary's publisher service sends `Create` activities to all followers whenever a new Stream is created.  The object type is determined by the Stream's Template. | When Emissary receives a "Create" activity, it adds a new message to that user's Inbox.  If the activity mentions (tags) the recipient, or replies to one of the recipient's own posts, Emissary also creates a Notification for the recipient — regardless of whether they follow the sender. |
| [Delete](https://www.w3.org/TR/activitypub/#delete-activity-outbox)/* | Emissary's publisher service sends a `Delete` activity to all followers whenever a Stream is unpublished. | When Emissary receives a `Delete` activity, it soft-deletes the corresponding message from the User's inbox. |
| [Dislike](https://www.w3.org/TR/activitystreams-vocabulary/#dfn-dislike) | Emissary sends a `Dislike` activity to a remote Inbox whenever a person responds NEGATIVELY to an external post. | User inboxes do not process inbound `Dislike` activities. Configured Stream actors can receive and rebroadcast them. |
| [Follow](https://www.w3.org/TR/activitypub/#follow-activity-outbox) | Emissary sends a `Follow` activity to a remote Inbox whenever a person requests to follow another ActivityPub Actor. | When Emissary receives a `Follow` activity, it validates the request, creates a new `Follower` record in the user's inbox, sends a corresponding `Accept` message to the originating server, and creates a Notification for the followed user.  Emissary does not currently allow user's to manually approve/disapprove follow requests. |
| [Like](https://www.w3.org/TR/activitystreams-vocabulary/#dfn-like) | Emissary sends a `Like` activity to a remote Inbox whenever a person responds POSITIVELY to an external post. | When Emissary receives a `Like` activity, creates a new `Response` record for the corresponding Stream.  If the liked object is one of the recipient's own posts, Emissary also creates a Notification for the recipient. |
| [Undo](https://www.w3.org/TR/activitypub/#undo-activity-outbox)/Block | Emissary sends an `Undo` activity whenever a user deletes or un-publishes a Block record in their profile. | Emissary does not currently process inbound `Undo` activities linked to `Block` on user inboxes. |
| [Undo](https://www.w3.org/TR/activitypub/#undo-activity-outbox)/Dislike | Emissary sends an `Undo` activity whenever a user deletes a NEGATIVE `Response` record in their profile. | When Emissary receives an `Undo` activity linked to a `Dislike`, it deletes the corresponding `Response` record from that user's profile. |
| [Undo](https://www.w3.org/TR/activitypub/#undo-activity-outbox)/Follow | Emissary sends an `Undo` activity whenever a user deletes a `Following` record in their profile. | When Emissary receives an `Undo` activity linked to a follow request, it deletes the corresponding `Follower` record from that user's profile. |
| [Undo](https://www.w3.org/TR/activitypub/#undo-activity-outbox)/Like | Emissary sends an `Undo` activity whenever a user deletes a POSITIVE `Response` record in their profile. | When Emissary receives an `Undo` activity linked to a `Like`, it deletes the corresponding `Response` record from that user's profile. |
| [Update](https://www.w3.org/TR/activitypub/#update-activity-outbox)/* | Emissary's publisher service sends an `Update` activity whenever a currently-published Stream is published again. | When Emissary receives an `Update` activity, it updates the corresponding message in that user's Inbox. |
| [Reject](https://www.w3.org/TR/activitystreams-vocabulary/#dfn-reject)/Follow | Emissary does not currently send `Reject` activities for follow requests. | Emissary does not currently process inbound follow rejections; a declined follow remains pending. |


## Notifications

Modeled on Mastodon, Emissary maintains a recipient-centric **Notification** for each inbound event that involves a local user: being **mentioned** (tagged) in a post, being **replied to**, having one's own content **liked / disliked / announced (boosted)**, and being **followed**.  Notifications are created regardless of whether the recipient follows the sender, and are filtered by the recipient's block/mute Rules.  An inbound `Undo` or `Delete` retracts the corresponding notification.

Notifications are delivered in real time to open browser tabs via **Server-Sent Events**, and to closed tabs via **Web Push** ([RFC 8291](https://datatracker.ietf.org/doc/html/rfc8291)), using a per-domain VAPID keypair.  They are also exposed through the Mastodon `/api/v1/notifications` endpoints (Dislike has no Mastodon equivalent and is omitted from that API).


## WebFinger

Emissary supports but does not require [WebFinger protocol](https://webfinger.net).  Every Emissary instance includes a **WebFinger server** that provides the publicly-available metadata about the people on that server, and is a **WebFinger client** that can use WebFinger to look up metadata from remote servers.

WebFinger also discovers Stream, Domain, Search, and Application actors, and advertises [FEP-3b86 Activity Intents](https://w3id.org/fep/3b86) where available.

## Additional Federation Features

* [ActivityPub](https://www.w3.org/TR/activitypub) actors — Users, Streams, Domains, saved Searches, and the Application actor publish profiles, inboxes, outboxes, and followable collections.
* [ActivityPub HTTP Signatures](https://www.w3.org/wiki/ActivityPub#Signing) — Inbound activities are signature-verified and outbound activities are signed per recipient.
* [NodeInfo 2.0](https://nodeinfo.diaspora.software/schema.html#schema-version-20) and [NodeInfo 2.1](https://nodeinfo.diaspora.software/schema.html#schema-version-21) — Emissary publishes both server metadata schemas.
* [MLS over ActivityPub](https://swicg.github.io/activitypub-e2ee/mls) — Users publish KeyPackages and receive MLS direct messages through dedicated collections and SSE topics.
* [ActivityPub Add and Remove](https://w3id.org/fep/7888) — Inbound activities can backfill conversation context.
* [ActivityPub Move](https://www.w3.org/TR/activitystreams-vocabulary/#dfn-move) — User inboxes process inbound account-migration activities.
* [ActivityPub actor updates](https://www.w3.org/TR/activitystreams-vocabulary/#dfn-update) — Inbound Person, Application, Group, Organization, and Service updates refresh actor profiles.
* [FEP-e232: Object Links](https://w3id.org/fep/e232) — Quote links in ActivityPub `Link` tags are consumed when resolving threads.

## RSS and Extensions

Emissary can read and write feeds in [RSS 2.0](https://en.wikipedia.org/wiki/RSS), [Atom](https://en.wikipedia.org/wiki/Atom_(web_standard)), and [JSONFeed](https://www.jsonfeed.org) formats.  The specific format is auto-negotiated, preferring: JSONFeed, then Atom, then RSS.

**Creating Feeds:** The `view-feed` step adds an RSS feed to any Stream, listing its children 

**Reading Feeds:** Users can follow any feed on the Internet by entering the site's URL into the "Follow" dialog.  

## MicroFormats

Emissary's default templates all include standard [MicroFormats](https://indieweb.org/microformats) for all available data points.

Emissary can also parse MicroFormats as a feed when following a URL.

## Mastodon API

Emisary implements a subset of the [Mastodon API](https://docs.joinmastodon.org/api/), allowing third-party Mastodon clients to interact with Emissary for all features commonly supported by both Emissary and Mastodon.

## FEPs

[Fediverse Enhancement Proposals](https://w3id.org/fep) extend ActivityPub and ActivityStreams.

### Implemented

* [FEP-3b86: Activity Intents](https://w3id.org/fep/3b86) — WebFinger advertises follow, like, announce, create, and reply intents.
* [FEP-d556: Server-Level Actor Discovery via WebFinger](https://w3id.org/fep/d556) — WebFinger exposes the server-level actor.
* [FEP-f1d5: NodeInfo 2.0](https://w3id.org/fep/f1d5) — NodeInfo 2.0 metadata is published.
* [FEP-0151: NodeInfo 2.1](https://w3id.org/fep/0151) — NodeInfo 2.1 metadata is published.
* [FEP-2677: Identifying the Application Actor](https://w3id.org/fep/2677) — The Application actor is published and discoverable.
* [FEP-67ff: FEDERATION.md](https://w3id.org/fep/67ff) — This document follows the FEDERATION.md convention.

### Partial or dormant

* [FEP-1b12: Group Federation](https://w3id.org/fep/1b12) — Stream rebroadcasting exists, but no shipped template defines a Stream actor.
* [FEP-7888: Demystifying the `context` Property](https://w3id.org/fep/7888) — Inbound Add and Remove activities can backfill conversation context.
* [FEP-e232: Object Links](https://w3id.org/fep/e232) — Quote links are consumed from ActivityPub Link tags.

### Not currently implemented

* [FEP-C648: Blocked Collection](https://w3id.org/fep/c648) — Legacy `/pub/blocked` URLs may be minted, but the collection is not served.

## Additional Documentation

These federation documents provide useful comparisons and implementation guidance:

* Mastodon - https://github.com/mastodon/mastodon/blob/main/FEDERATION.md
* Streams - https://codeberg.org/streams/streams/src/branch/dev/FEDERATION.md
* WriteFreely - https://github.com/writefreely/documentation/blob/master/writer/federation.md
