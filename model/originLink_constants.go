package model

// OriginTypePrimary identifies an original post by an author (not a reply, announce, like, or dislike)
const OriginTypePrimary = "PRIMARY"

// OriginTypeReply identifies a link that was retrieved because of a "Reply" to an existing post
const OriginTypeReply = "REPLY"

// OriginTypeAnnounce identifies a link that was retrieved because of a "Announce" of an existing post
const OriginTypeAnnounce = "ANNOUNCE"

// OriginTypeBoost identifies a link that was retrieved because of a "Like" of an existing post
const OriginTypeLike = "LIKE"

// OriginTypeBoost identifies a link that was retrieved because of a "Dislike" of an existing post
const OriginTypeDislike = "DISLIKE"
