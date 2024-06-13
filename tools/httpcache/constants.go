package httpcache

// metadataMarker is used to mark a cache entry that contains response metadata
const metadataMarker = "::META"

// headSeparator is used to separate URL from the header values in a cache key
const headSeparator = "::HEAD::"
