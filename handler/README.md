# handler

This package contains Emissary's HTTP handlers — the functions wired to routes that turn an incoming request into a response. Handlers here cover the web UI, the JSON API, file/attachment serving, OAuth, and sign-in. Protocol-specific handlers live in subdirectories: [activitypub_user](activitypub_user/), [activitypub_stream](activitypub_stream/), [activitypub_domain](activitypub_domain/), [activitypub_search](activitypub_search/), and the Mastodon-compatible API under [mastodon](mastodon/), plus third-party integrations like [stripe](stripe/) and [unsplash](unsplash/).

Handlers stay thin: they parse the request, call into the [service](../service/) and [build](../build/) layers, and write the result. See the [project README](../README.md) for the big picture.
