# realtime

This package pushes live updates to connected browsers. The `Broker` is a singleton that tracks which clients are currently attached and broadcasts events to them, so that a change made in one place (a new message, an updated stream) appears in open browser tabs without a page reload. It is the server side of Emissary's real-time UI.

See the [project README](../README.md) for the big picture.
