package asnormalizer

// maxDepth is how many documents a chain of nested loads may hold before the next load returns a stub.
// Real traffic needs two levels: a post, then its author, whose own links load at the second level.
const maxDepth = 3
