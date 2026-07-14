# server

This package wires up the running Emissary server. Its `Factory` owns the server-level services (those shared across every domain) and produces an individual domain factory for each domain the server hosts — the seam between one process and the many domains it serves. This is where server-wide configuration is turned into live, per-domain machinery.

See the [project README](../README.md) for the big picture, and [config](../config/) for the configuration this factory consumes.
