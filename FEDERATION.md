# Federation

Emissary supports many standard ways of following and sharing content.  The goal is to accept the highest-fidelity connections from as many sites as possible, whatever specific protocols they are using at the time.  In many cases, Emissary will support several competing standards, defaulting to its preferred methods, but accepting connections in whatever form they are offered.

## ActivityPub

The [ActivityPub](https://activitypub.rocks) standard is currently the most common and popular protocol for sharing information between users on various websites.  Emissary currently supports the following Activities defined in [the ActivityPub spec](https://www.w3.org/TR/activitypub).  Other unrecognized activities are ignored.

| Activity | Sending | Receiving |
| -------- | ------- | --------- |
| [Accept](https://www.w3.org/TR/activitypub/#accept-activity-inbox)/Follow | When Emissary receives a follow request, it adds a new "Follower" record and sends a corresponding `Accept` activity to the original server. | When Emissary receives an `Accept` activity tied to a `Follow` activity, it mark the corresponding `Following` record as active. Other forms of `Accept` are ignored.|
| [Block](https://www.w3.org/TR/activitystreams-vocabulary/#dfn-block) | Emissary sends a `Block` activity to all followers whenever a user creates a Block in their profile that is shared publicly. | When Emissary receives a `Block` activity from a remote actor it follows, it creates a block recommendation for the current user that includes the reason the remote actor provided for the block. |
| [Create](https://www.w3.org/TR/activitypub/#create-activity-inbox)/* | Emissary's publisher service sends `Create` activities to all followers whenever a new Stream is created.  The object type is determined by the Stream's Template. | When Emissary receives a "Create" activity, it adds a new message to that user's Inbox. |
| [Delete](https://www.w3.org/TR/activitypub/#delete-activity-outbox)/* | Emissary's publisher service sends a `Delete` activity to all followers whenever a Stream is unpublished. | When Emissary receives a `Delete` activity, it soft-deletes the corresponding message from the User's inbox. |
| [Dislike](https://www.w3.org/TR/activitystreams-vocabulary/#dfn-dislike) | Emissary sends a `Dislike` activity to a remote Inbox whenever a person responds NEGATIVELY to an external post. | When Emissary receives a `Dislike` activity, creates a new `Response` record for the corresponding Stream. |
| [Follow](https://www.w3.org/TR/activitypub/#follow-activity-outbox) | Emissary sends a `Follow` activity to a remote Inbox whenever a person requests to follow another ActivityPub Actor. | When Emissary receives a `Follow` activity, it validates the request, creates a new `Follower` record in the user's inbox, and then sends a corresponding `Accept` message to the originating server.  Emissary does not currently allow user's to manually approve/disapprove follow requests. |
| [Like](https://www.w3.org/TR/activitystreams-vocabulary/#dfn-like) | Emissary sends a `Like` activity to a remote Inbox whenever a person responds POSITIVELY to an external post. | When Emissary receives a `Like` activity, creates a new `Response` record for the corresponding Stream. |
| [Undo](https://www.w3.org/TR/activitypub/#undo-activity-outbox)/Block | Emissary sends an `Undo` activity whenever a user deletes or un-publishes a Block record in their profile. | When Emissary receives an `Undo` activity linked to a `Block`, it deletes the corresponding `Block` recommendation record from that user's profile. |
| [Undo](https://www.w3.org/TR/activitypub/#undo-activity-outbox)/Dislike | Emissary sends an `Undo` activity whenever a user deletes a NEGATIVE `Response` record in their profile. | When Emissary receives an `Undo` activity linked to a `Dislike`, it deletes the corresponding `Response` record from that user's profile. |
| [Undo](https://www.w3.org/TR/activitypub/#undo-activity-outbox)/Follow | Emissary sends an `Undo` activity whenever a user deletes a `Following` record in their profile. | When Emissary receives an `Undo` activity linked to a follow request, it deletes the corresponding `Follower` record from that user's profile. |
| [Undo](https://www.w3.org/TR/activitypub/#undo-activity-outbox)/Like | Emissary sends an `Undo` activity whenever a user deletes a POSITIVE `Response` record in their profile. | When Emissary receives an `Undo` activity linked to a `Like`, it deletes the corresponding `Response` record from that user's profile. |
| [Update](https://www.w3.org/TR/activitypub/#update-activity-outbox)/* | Emissary's publisher service sends an `Update` activity whenever a currently-published Stream is published again. | When Emissary receives an `Update` activity, it updates the corresponding message in that user's Inbox.


## WebFinger

Emissary supports but does not require [WebFinger protocol](https://webfinger.net).  Every Emissary instance includes a WebFinger server that provides the publicly-available metadata about the people on that server.

## RSS and Extensions

Emissary can read and write feeds in [RSS 2.0](https://en.wikipedia.org/wiki/RSS), [Atom](https://en.wikipedia.org/wiki/Atom_(web_standard\)), and [JSONFeed](https://www.jsonfeed.org) formats.  The specific format is auto-negotiated, preferring: JSONFeed, then Atom, then RSS.

**Creating Feeds:** The `view-feed` step adds an RSS feed to any Stream, listing its children 

**Reading Feeds:** Users can follow any feed on the Internet by entering the site's URL into the "Follow" dialog.  

## WebSub

Emissary sends and receives real-time feed updates via [WebSub protocol](https://www.w3.org/TR/websub/).  The publisher service works as its own WebSub hub, sending updates whenever a Stream us published or republished.


## WebMentions

Emissary's publisher service sends WebMentions whenever a Stream is published or re-published.

Default templates also include meta-data that points to Emissary's WebMention receiver, which receives WebMentions from external servers, which are stored a "Mentions" in Emissary's database and are accessible to all Stream Templates.

## MicroFormats

Emissary's default templates all include standard [MicroFormats](https://indieweb.org/microformats) for all available data points.

Emissary can also parse MicroFormats as a feed when following a URL.


## Work In Progress

This is a placeholder for writing FEDERATION.md documentation, similar to the entries listed here:
https://socialhub.activitypub.rocks/t/guide-for-new-activitypub-implementers/479#federationmd-25

* Mastodon - https://github.com/mastodon/mastodon/blob/main/FEDERATION.md
* Streams - https://codeberg.org/streams/streams/src/branch/dev/FEDERATION.md
* WriteFreely - https://github.com/writefreely/documentation/blob/master/writer/federation.md