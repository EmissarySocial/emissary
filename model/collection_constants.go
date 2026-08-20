package model

import "github.com/benpate/hannibal/vocab"

// CollectionParentTypeUser identifies a Collection that belongs to a User
const CollectionParentTypeUser = "User"

// CollectionParentTypeStream identifies a Collection that belongs to a Stream
const CollectionParentTypeStream = "Stream"

// CollectionTypeContext is the type of collection that is used to group messages that are part of the same conversation thread.
// https://www.w3.org/TR/activitystreams-vocabulary/#dfn-context
const CollectionTypeContext = "Context"

// CollectionTypeReplies is the type of collection that is used to group messages that are replies to a specific message.
// https://www.w3.org/TR/activitystreams-vocabulary/#dfn-replies
const CollectionTypeReplies = "Replies"

// CollectionTypeLikes is the type of collection that groups the actors who "Like" a specific message.
// https://www.w3.org/TR/activitystreams-vocabulary/#dfn-likes
const CollectionTypeLikes = "Likes"

// CollectionTypeDislikes is the type of collection that groups the actors who "Dislike" a specific message.
const CollectionTypeDislikes = "Dislikes"

// CollectionTypeShares is the type of collection that groups the actors who "Announce" (share) a specific message.
// https://www.w3.org/TR/activitystreams-vocabulary/#dfn-shares
const CollectionTypeShares = "Shares"

// CollectionTypeForResponse maps a response Type to the Collection Type that stores it.
// It returns "" for response types not projected into a per-Stream collection.
func CollectionTypeForResponse(responseType string) string {
	switch responseType {
	case vocab.ActivityTypeLike:
		return CollectionTypeLikes
	case vocab.ActivityTypeDislike:
		return CollectionTypeDislikes
	case vocab.ActivityTypeAnnounce:
		return CollectionTypeShares
	default:
		return ""
	}
}
