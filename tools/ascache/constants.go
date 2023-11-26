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

// RelationTypeAnnounce labels a document that is an "Announce" of another
// document in the cache.  This value mirrors the ActivityStream "Announce" type
const RelationTypeAnnounce = "Announce"

// RelationTypeReply labels a document that is a reply to another document in the cache
const RelationTypeReply = "Reply"

// RelationTypeLike labels a document that is a "Like" of another
// document in the cache.  This value mirrors the ActivityStream "Like" type
const RelationTypeLike = "Like"

// RelationTypeDislike labels a document that is a "Dislike" of another
// document in the cache.  This value mirrors the ActivityStream "Dislike" type
const RelationTypeDislike = "Dislike"
