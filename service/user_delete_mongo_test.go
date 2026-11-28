package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// newUserDeleteTestService returns a User service wired with every service that Delete calls
func newUserDeleteTestService() User {

	return User{
		annotationService:     &Annotation{},
		domainService:         &Domain{},
		folderService:         &Folder{},
		followerService:       &Follower{},
		followingService:      &Following{},
		keyService:            &EncryptionKey{},
		newsFeedService:       &NewsFeed{},
		outboxService:         &Outbox{},
		outbox2Service:        &Outbox2{},
		responseService:       &Response{},
		ruleService:           &Rule{},
		searchResultService:   &SearchResult{},
		streamService:         &Stream{},
		userConnectionService: &UserConnection{},
	}
}

// TestUserDelete_DeletesOnlyTheOwnersAnnotations runs Delete against a real MongoDB, and confirms
// that the deleted User's notes go with them while another User's notes on the same URL stay.
func TestUserDelete_DeletesOnlyTheOwnersAnnotations(t *testing.T) {

	_, session := newReplicaSetSession(t)
	service := newUserDeleteTestService()

	owner := model.NewUser()
	owner.UserID = primitive.NewObjectID()
	owner.Username = "owner"
	owner.ProfileURL = "https://example.test/@owner"
	require.NoError(t, session.Collection("User").Save(&owner, "test setup"))

	otherUserID := primitive.NewObjectID()

	insert := func(userID primitive.ObjectID, url string) {
		annotation := model.NewAnnotation()
		annotation.UserID = userID
		annotation.URL = url
		annotation.Type = model.AnnotationTypeDocument
		annotation.Content = "private note"
		require.NoError(t, session.Collection("Annotation").Save(&annotation, "test setup"))
	}

	insert(owner.UserID, "https://example.test/notes/1")
	insert(owner.UserID, "https://example.test/notes/2")
	insert(otherUserID, "https://example.test/notes/1")

	require.NoError(t, service.Delete(session, &owner, "test"))

	count := func(userID primitive.ObjectID) int64 {
		result, err := session.Collection("Annotation").Count(exp.Equal("userId", userID).AndEqual("deleteDate", 0))
		require.NoError(t, err)
		return result
	}

	require.Zero(t, count(owner.UserID))
	require.Equal(t, int64(1), count(otherUserID))
}

// TestUserDelete_ClearsOutbox2BeforeStreams verifies that deleting a User removes their Outbox2 activities
// before their Streams, and that deleting those Streams queues no delivery
func TestUserDelete_ClearsOutbox2BeforeStreams(t *testing.T) {

	test := newStreamDeleteTest(t)
	service := newUserDeleteTestService()
	service.streamService = &test.service

	owner := model.NewUser()
	owner.UserID = primitive.NewObjectID()
	owner.Username = "owner"
	owner.ProfileURL = "https://example.test/@owner"
	require.NoError(t, test.session.Collection("User").Save(&owner, "test setup"))

	otherUserID := primitive.NewObjectID()
	test.saveOutbox2Activity(t, model.ActorTypeUser, otherUserID, "https://example.test/notes/1", vocab.ActivityTypeCreate)

	stream := test.saveStream(t, owner.UserID, owner.UserID, true)
	test.saveOutbox2Activity(t, model.ActorTypeUser, owner.UserID, stream.URL, vocab.ActivityTypeCreate)

	require.NoError(t, service.Delete(test.session, &owner, "test"))

	require.Zero(t, test.deletesSent(t), "deleting a User sends no Delete for each of their Streams")
	require.Zero(t, test.countOutbox2(t, model.ActorTypeUser, owner.UserID))
	require.Equal(t, int64(1), test.countOutbox2(t, model.ActorTypeUser, otherUserID))
}

// TestUserDelete_LeavesTheContextsOfThreadsTheyRepliedTo pins that deleting an account removes its
// replies from other Users' threads, and leaves each thread's Context in place (BUG-312)
func TestUserDelete_LeavesTheContextsOfThreadsTheyRepliedTo(t *testing.T) {

	test := newStreamDeleteTest(t)
	service := newUserDeleteTestService()
	service.streamService = &test.service

	bob := model.NewUser()
	bob.UserID = primitive.NewObjectID()
	bob.Username = "bob"
	bob.ProfileURL = "https://example.test/@bob"
	require.NoError(t, test.session.Collection("User").Save(&bob, "test setup"))

	aliceID := primitive.NewObjectID()
	frankID := primitive.NewObjectID()

	aliceRoot := test.saveStream(t, aliceID, aliceID, true)
	aliceContext := test.saveContext(t, &aliceRoot, aliceID)

	frankRoot := test.saveStream(t, frankID, frankID, true)
	frankContext := test.saveContext(t, &frankRoot, frankID)

	// Bob replies in both threads, and starts one of his own
	toAlice := test.saveReply(t, bob.UserID, &aliceRoot)
	test.addItem(t, &aliceContext, toAlice.ActivityPubURL())

	toFrank := test.saveReply(t, bob.UserID, &frankRoot)
	test.addItem(t, &frankContext, toFrank.ActivityPubURL())

	bobRoot := test.saveStream(t, bob.UserID, bob.UserID, true)
	bobContext := test.saveContext(t, &bobRoot, bob.UserID)

	require.NoError(t, service.Delete(test.session, &bob, "test"))

	require.Equal(t, []string{aliceRoot.ActivityPubURL()}, test.contextURLs(t, aliceContext))
	require.Equal(t, []string{frankRoot.ActivityPubURL()}, test.contextURLs(t, frankContext))
	require.Nil(t, test.contextURLs(t, bobContext), "a thread the deleted User started goes with them")
}
