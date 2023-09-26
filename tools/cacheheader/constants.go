package cacheheader

/******************************************
 * HTTP Headers
 ******************************************/

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Age
const HeaderAge = "age"

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control
const HeaderCacheControl = "Cache-Control"

/******************************************
 * Cache-Control Directives
 ******************************************/

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control#max-age
const DirectiveMaxAge = "max-age"

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control#s-max-age
const DirectiveSMaxAge = "s-maxage"

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control#no-cache
const DirectiveNoCache = "no-cache"

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control#no-store
const DirectiveNoStore = "no-store"

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control#no-transform
const DirectiveNoTransform = "no-transform"

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control#must-revalidate
const DirectiveMustRevalidate = "must-revalidate"

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control#proxy-revalidate
const DirectiveProxyRevalidate = "proxy-revalidate"

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control#must-understand
const DirectiveMustUnderstand = "must-understand"

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control#private
const DirectivePrivate = "private"

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control#public
const DirectivePublic = "public"

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control#immutable
const DirectiveImmutable = "immutable"

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control#stale-while-revalidate
const DirectiveStaleWhileRevalidate = "stale-while-revalidate"

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control#stale-if-error
const DirectiveStaleIfError = "stale-if-error"
