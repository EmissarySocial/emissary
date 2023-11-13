package ascache

// CacheModeReadWrite represents a cache configuration that reads from and writes to the cache
const CacheModeReadWrite = "READWRITE"

// CacheModeReadWrite represents a cache configuration that only reads from the cache.
// It does not update the cache with new values.
const CacheModeReadOnly = "READONLY"

// CacheModeReadWrite represents a cache configuration that only writes to the cache.
// It does not search for existing cached values
const CacheModeWriteOnly = "WRITEONLY"

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Age
const headerAge = "Age"

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Date
const headerDate = "Date"

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Expires
const headerExpires = "Expires"

// Custom header used by Hannibal to indicate that the response was cached
const headerHannibalCache = "X-Hannibal-Cache"

// Custom header used by Hannibal to indicate the date that the cached value was saved
const headerHannibalCacheDate = "X-Hannibal-Cache-Date"
