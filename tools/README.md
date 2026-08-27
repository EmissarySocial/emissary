# tools

This directory is a home for small, self-contained helper packages — libraries just big enough to deserve their own package, but not big enough (yet) to warrant a separate repository. They cover a wide range: caching (`ascache`, `httpcache`, `cacheheader`), ActivityStreams helpers (`asnormalizer`, `asstrict`, `ashash`, `ascontextmaker`), crypto (`hmac`), media and HTTP utilities (`headers`, `s3uri`, `striputm`), and many more.

Some of these may eventually graduate into their own repositories; for now, keeping them here avoids the cost of another external dependency during active development. See the [project README](../README.md) for the big picture.
