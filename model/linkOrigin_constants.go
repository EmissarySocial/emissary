package model

// OriginTypeActivityPub identifies a link that was created by an ActivityPub push
const OriginTypeActivityPub = "ACTIVITYPUB"

// OriginTypeInternal identifies a link was created by this server
const OriginTypeInternal = "INTERNAL"

// OriginTypePoll identifies a link that was polled from an RSS source
const OriginTypePoll = "POLL"

// OriginTypeRSSCloud identifies a link that was created by an RSS-Cloud push
const OriginTypeRSSCloud = "RSS-CLOUD"

// OriginTypeWebMention identifies a link that was created by a WebMention push
const OriginTypeWebMention = "WEBMENTION"

// OriginTypeWebSub identifies a link that was created by a WebSub push
const OriginTypeWebSub = "WEBSUB"
