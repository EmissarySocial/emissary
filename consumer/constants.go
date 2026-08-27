package consumer

// maxCrawlDepth bounds the reply-tree crawlers (CrawlUpReplyTree, CrawlDownReplyTree).
// The reply graph is remote-controlled data, so a cycle (self-replies, mutual replies,
// an ancestor listed in a "replies" collection) or an endless chain of generated
// "inReplyTo" URLs would otherwise breed queue tasks forever.  Each crawl task carries
// a "depth" argument that increments as the crawl recurses, and stops here.
const maxCrawlDepth = 32
