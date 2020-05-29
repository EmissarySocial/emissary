package model

// StreamSourceType enumerates all of the possible values for a stream.Source variable
type StreamSourceType string

// StreamSourceActivityPub identifies a Stream that originated on an external ActivityPub server
const StreamSourceActivityPub StreamSourceType = "ACTIVITYPUB"

// StreamSourceEmail identifies a Stream that originated on an external Email server
const StreamSourceEmail StreamSourceType = "EMAIL"

// StreamSourceRSS identifies a Stream that originated on an external RSS feed
const StreamSourceRSS StreamSourceType = "RSS"

// StreamSourceSystem identifies a Stream that originated on this server
const StreamSourceSystem StreamSourceType = "SYSTEM"

// StreamSourceTwitter identifies a Stream that originated on Twitter
const StreamSourceTwitter StreamSourceType = "TWITTER"
