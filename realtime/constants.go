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
