package service

import (
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SaveTaggedNewsItem adds a post to a User's home timeline because they follow one of its #hashtags
func (service *Following) SaveTaggedNewsItem(session data.Session, userID primitive.ObjectID, tagName string, document streams.Document) error {

	// A stand-in Following names the tag as the post's origin. Its zero FollowingID and
	// FolderID keep it apart from real follows, so unfollowing an account never removes it.
	following := model.NewFollowing()
	following.FollowingID = primitive.NilObjectID
	following.UserID = userID
	following.URL = service.host + "/tags/" + url.PathEscape(tagName)
	following.Label = "#" + tagName

	// SaveNewsItem applies the User's blocks and mutes, and merges duplicates by URL
	if err := service.SaveNewsItem(session, &following, document, model.OriginTypeHashtag); err != nil {
		return derp.Wrap(err, "service.Following.SaveTaggedNewsItem", "Saving news item", userID, tagName)
	}

	// Tagged and bagged
	return nil
}
