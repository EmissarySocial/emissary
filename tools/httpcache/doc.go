// Package httpcache caches outbound HTTP responses.
//
// HTTPMiddleware is an http.RoundTripper that sits in a client's transport chain: it answers
// a repeated GET from the cache instead of re-fetching it, which matters most for
// ActivityPub, where rendering one page can touch the same remote actor many times.
//
// Storage is pluggable behind the HTTPCache adapter.  Two are shipped -- an in-process Otter
// cache and a shared Redis cache -- so a single server and a fleet can use the same
// middleware.  Only cacheable responses are stored, and a TTL is applied per entry.
package httpcache
