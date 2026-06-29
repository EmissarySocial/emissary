# config

This package defines Emissary's server-level configuration — the set of domains served by this instance, plus the storage, database, and command-line settings that the server reads at startup. `Config` is the top-level structure; the accessors and loaders here parse it from its various sources and supply sensible defaults.

See the [project README](../README.md) for the big picture.
