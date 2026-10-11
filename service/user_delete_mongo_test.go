package service

import (
	"context"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// newUserDeleteTestService returns a User service wired with every service that Delete calls.
// The webhook session refuses to open, and closes the returned channel when Delete asks for it.
func newUserDeleteTestService() (User, <-chan struct{}) {

	webhookCalled := make(chan struct{})

	webhookService := &Webhook{
		newSession: func(time.Duration) (data.Session, context.CancelFunc, error) {
			close(webhookCalled)
			return nil, nil, derp.Internal("test", "Webhooks are not delivered in this test")
		},
	}

	return User{
		annotationService:     &Annotation{},
		folderService:         &Folder{},
		followerService:       &Follower{},
		followingService:      &Following{},
		keyService:            &EncryptionKey{},
		newsFeedService:       &NewsFeed{},
		outboxService:         &Outbox{},
		responseService:       &Response{},
		ruleService:           &Rule{},
		searchResultService:   &SearchResult{},
		streamService:         &Stream{},
		userConnectionService: &UserConnection{},
		webhookService:        webhookService,
	}, webhookCalled
}

// TestUserDelete_DeletesOnlyTheOwnersAnnotations runs Delete against a real MongoDB, and confirms
// that the deleted User's notes go with them while another User's notes on the same URL stay.
func TestUserDelete_DeletesOnlyTheOwnersAnnotations(t *testing.T) {

	_, session := newReplicaSetSession(t)
	service, webhookCalled := newUserDeleteTestService()

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

	// Wait for the user:delete webhook, so its goroutine does not outlive the test
	select {
	case <-webhookCalled:
	case <-time.After(5 * time.Second):
		t.Fatal("Delete never sent the user:delete webhook")
	}

	count := func(userID primitive.ObjectID) int64 {
		result, err := session.Collection("Annotation").Count(exp.Equal("userId", userID).AndEqual("deleteDate", 0))
		require.NoError(t, err)
		return result
	}

	require.Zero(t, count(owner.UserID))
	require.Equal(t, int64(1), count(otherUserID))
}
