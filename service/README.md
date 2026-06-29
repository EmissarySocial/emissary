# service

This package is Emissary's business-logic layer. Each service (Stream, User, Follower, Following, Conversation, Inbox, Outbox, and many more) manages all interactions with one [model](../model/) collection — loading, saving, validating, and coordinating the side effects of a change. Services are where the rules live; [handlers](../handler/) and [builders](../build/) call into them rather than touching the database directly. Import and export logic for each object lives alongside its service here too.

Subdirectories hold pluggable integrations: [providers](providers/) for external services (e.g. Stripe Connect) and [geocoder](geocoder/) for address lookups. See the [project README](../README.md) for the big picture.
