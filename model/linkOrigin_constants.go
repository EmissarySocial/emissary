package model

/*
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
*/
// OriginTypeDirect identifies a link that was created by directly by the Author.
const OriginTypeDirect = "DIRECT"

// OriginTypeReply identifies a link that was retrieved because of a "Reply" to an existing post
const OriginTypeReply = "REPLY"

// OriginTypeAnnounce identifies a link that was retrieved because of a "Announce" of an existing post
const OriginTypeAnnounce = "ANNOUNCE"

// OriginTypeBoost identifies a link that was retrieved because of a "Like" of an existing post
const OriginTypeLike = "LIKE"

// OriginTypeBoost identifies a link that was retrieved because of a "Dislike" of an existing post
const OriginTypeDislike = "DISLIKE"
