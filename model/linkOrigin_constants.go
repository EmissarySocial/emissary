package model

// OriginTypeActivityPub identifies a link that was created by an ActivityPub push
const OriginTypeActivityPub = "ACTIVITYPUB"

// OriginTypePoll identifies a link that was polled from an RSS source
const OriginTypePoll = "POLL"

// OriginTypeRSSCloud identifies a link that was created by an RSS-Cloud push
const OriginTypeRSSCloud = "RSS-CLOUD"

// OriginTypeWebMention identifies a link that was created by a WebMention push
const OriginTypeWebMention = "WEBMENTION"

// OriginTypeWebSub identifies a link that was created by a WebSub push
const OriginTypeWebSub = "WEBSUB"

// OriginTypeMention identifies a link that was created by a mention of an existing post
const OriginTypeMention = "MENTION"

// OriginTypeReply identifies a link that was created by a reply to an existing post
const OriginTypeReply = "REPLY"

// OriginTypeBoost identifies a link that was created by a boost of an existing post
const OriginTypeBoost = "BOOST"
