package service

import (
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/asnormalizer"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Tag Follows
 *
 * Users follow #hashtags here. When a public post with a
 * followed tag arrives or is published, a background task
 * (TaskDeliverToTagFollowers) adds it to each follower's
 * home timeline.
 ******************************************/

// TaskDeliverToTagFollowers is the queue task that adds a tagged post to its followers' home timelines
const TaskDeliverToTagFollowers = "DeliverToTagFollowers"

// TagFollow manages the #hashtags that Users follow
type TagFollow struct {
	hostname string
	queue    *queue.Queue
}

// NewTagFollow returns a fully initialized TagFollow service
func NewTagFollow() TagFollow {
	return TagFollow{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service
func (service *TagFollow) Refresh(factory *Factory) {
	service.hostname = factory.Hostname()
	service.queue = factory.Queue()
}

/******************************************
 * Common Data Methods
 ******************************************/

// collection returns the TagFollow collection for the provided database session
func (service *TagFollow) collection(session data.Session) data.Collection {
	return session.Collection("TagFollow")
}

// Query returns the TagFollows that match the criteria
func (service *TagFollow) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.TagFollow, error) {
	result := make([]model.TagFollow, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

/******************************************
 * Custom Methods
 ******************************************/

// Follow makes the User follow the tag. Following a tag twice changes nothing.
func (service *TagFollow) Follow(session data.Session, userID primitive.ObjectID, name string) error {

	const location = "service.TagFollow.Follow"

	// RULE: The tag must have a name
	tagFollow := model.NewTagFollow()
	tagFollow.UserID = userID
	tagFollow.Name = TagFollowName(name)
	tagFollow.Value = model.ToToken(tagFollow.Name)

	if tagFollow.Value == "" {
		return derp.Validation("Hashtag name is required")
	}

	// Already following? Then we're done
	following, err := service.IsFollowing(session, userID, tagFollow.Value)

	if err != nil {
		return derp.Wrap(err, location, "Checking existing follow", tagFollow.Value)
	}

	if following {
		return nil
	}

	// Save the new follow
	if err := service.collection(session).Save(&tagFollow, "Followed"); err != nil {
		return derp.Wrap(err, location, "Saving TagFollow", tagFollow.Value)
	}

	// Once this commits, subscribe the server to the tag's relay and fill this User's timeline
	service.queueRelayTask(session, TaskSubscribeTagRelay, mapof.Any{"value": tagFollow.Value, "name": tagFollow.Name}, tagFollow.Value)
	service.queueRelayTask(session, TaskBackfillTagFollow, mapof.Any{"userId": userID.Hex(), "name": tagFollow.Name, "value": tagFollow.Value}, userID.Hex()+":"+tagFollow.Value)

	// Tag, you're it
	return nil
}

// Unfollow makes the User stop following the tag. Posts already in their timeline stay there.
func (service *TagFollow) Unfollow(session data.Session, userID primitive.ObjectID, name string) error {

	const location = "service.TagFollow.Unfollow"

	// Remove every record of this User following this tag
	value := model.ToToken(TagFollowName(name))
	criteria := exp.Equal("userId", userID).AndEqual("value", value)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Deleting TagFollow", name)
	}

	// Once this commits, drop the server's relay subscription if that was the last follower
	service.queueRelayTask(session, TaskUnsubscribeTagRelay, mapof.Any{"value": value}, value)

	// Tag, you're not it
	return nil
}

// IsFollowing returns TRUE if the User follows the tag with this normalized value
func (service *TagFollow) IsFollowing(session data.Session, userID primitive.ObjectID, value string) (bool, error) {

	count, err := service.collection(session).Count(notDeleted(exp.Equal("userId", userID).AndEqual("value", value)))

	if err != nil {
		return false, derp.Wrap(err, "service.TagFollow.IsFollowing", "Counting TagFollows", value)
	}

	return count > 0, nil
}

// IsFollowedByAnyone returns TRUE if any User here follows the tag with this normalized value
func (service *TagFollow) IsFollowedByAnyone(session data.Session, value string) (bool, error) {

	count, err := service.collection(session).Count(notDeleted(exp.Equal("value", value)))

	if err != nil {
		return false, derp.Wrap(err, "service.TagFollow.IsFollowedByAnyone", "Counting TagFollows", value)
	}

	return count > 0, nil
}

// QueryByUser returns the tags the User follows, newest first
func (service *TagFollow) QueryByUser(session data.Session, userID primitive.ObjectID, limit int64) ([]model.TagFollow, error) {
	return service.Query(session, exp.Equal("userId", userID), option.SortDesc("createDate"), option.MaxRows(limit))
}

// QueryByValues returns every follow of any of the normalized tag values
func (service *TagFollow) QueryByValues(session data.Session, values []string) ([]model.TagFollow, error) {

	if len(values) == 0 {
		return []model.TagFollow{}, nil
	}

	return service.Query(session, exp.In("value", values), option.Fields("userId", "name", "value"))
}

// QueueDelivery asks the queue, once this transaction commits, to add a post to the timelines of its
// #hashtags' followers. A local author (authorID) does not get their own post.
func (service *TagFollow) QueueDelivery(session data.Session, document streams.Document, authorID primitive.ObjectID) {

	// RULE: Untagged and non-public posts have no one to reach, so skip the task
	if !IsPublicPost(document) || len(DocumentTagValues(document)) == 0 {
		return
	}

	// The signature collapses the same post arriving in many inboxes into one task
	postcommit.Publish(
		session,
		service.queue,
		TaskDeliverToTagFollowers,
		mapof.Any{"url": document.ID(), "hostname": service.hostname, "authorId": authorID.Hex()},
		queue.WithSignature(TaskDeliverToTagFollowers+":"+service.hostname+":"+document.ID()),
	)
}

// queueRelayTask asks the queue to run a relay task once this transaction commits
func (service *TagFollow) queueRelayTask(session data.Session, name string, args mapof.Any, key string) {
	args["hostname"] = service.hostname

	// The key collapses repeats, so following and unfollowing in a burst runs each task once
	postcommit.Publish(session, service.queue, name, args, queue.WithSignature(name+":"+service.hostname+":"+key))
}

/******************************************
 * Helpers
 ******************************************/

// TagFollowName returns a tag's name without whitespace or the leading #
func TagFollowName(name string) string {
	return strings.TrimPrefix(strings.TrimSpace(name), "#")
}

// DocumentTagValues returns the normalized values of a document's #hashtags, skipping mentions and emoji
func DocumentTagValues(document streams.Document) []string {

	result := make([]string, 0)

	// Range walks the list; tag.Next() would read the ActivityStreams "next" property and stop after one
	for tag := range document.Tag().Range() {

		// A bare-string tag would need a fetch to read its name, so skip it (as DocumentMatchKeys does)
		if tag.IsString() || tag.Type() != vocab.LinkTypeHashtag {
			continue
		}

		if value := model.ToToken(tag.Name()); value != "" {
			result = append(result, value)
		}
	}

	return result
}

// IsPublicPost returns TRUE if a post's visibility is "public", from either a cached or a raw document
func IsPublicPost(document streams.Document) bool {

	// Cached posts keep only a "visibility" word, because the normalizer drops "to" and "cc"
	visibility := document.Get("visibility").String()

	if visibility == "" {
		visibility = asnormalizer.Visibility(document, "")
	}

	return visibility == "public"
}
