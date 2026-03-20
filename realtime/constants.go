package realtime

// TopicAll represents all SSE topics
const TopicAll = 0

// TopicUpdated is triggered when a Stream or User is updated
const TopicUpdated = 1

// TopicChildUpdated is triggered when the child of a Stream is updated
const TopicChildUpdated = 2

// TopicNewReplies is triggered when there are one or more new replies to an Inbox message
const TopicNewReplies = 3

// TopicImportProgress is triggered to report progress during an import operation
const TopicImportProgress = 4

// TopicFollowingUpdated is triggered when a Following record has a new status
const TopicFollowingUpdated = 4

// TopicInboxActivity is triggered when there is new activity in a User's Inbox
const TopicInboxActivity = 5

// TopicInboxActivity_DirectMessage is triggered when a new Direct Message is received
const TopicInboxActivity_DirectMessage = 6

// TopicInboxActivity_DirectMessage_MLS is triggered when a new Direct Message with mediaType "message/mls" is received
const TopicInboxActivity_DirectMessage_MLS = 7
