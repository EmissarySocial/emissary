package service

import (
	"context"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// These tests cover the visibility gate in User.WebFinger: a non-public profile must not resolve
// through WebFinger discovery, matching the hidden actor document and every sibling ActivityPub
// endpoint. They use a hand-built data.Collection fake (mirroring circle_test.go / response_test.go)
// because WebFinger only needs LoadByUsername, and data-mock cannot match the model's omitempty tags.

/******************************************
 * In-Memory Fakes
 ******************************************/

// userCollection is an in-memory data.Collection that holds a single model.User record,
// matched on the fields LoadByUsername + notDeleted build: username and deleteDate.
type userCollection struct {
	record model.User
	found  bool
}

// Context implements the interface, returning a background context
func (c *userCollection) Context() context.Context { return context.Background() }

// Count implements the data.Collection interface. Unused by these tests.
func (c *userCollection) Count(exp.Expression, ...option.Option) (int64, error) {
	return 0, derp.Internal("test", "unused")
}

// Query implements the data.Collection interface. Unused by these tests.
func (c *userCollection) Query(any, exp.Expression, ...option.Option) error {
	return derp.Internal("test", "unused")
}

// Iterator implements the data.Collection interface. Unused by these tests.
func (c *userCollection) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.Internal("test", "unused")
}

// Load copies the stored User into the target when the criteria match.
func (c *userCollection) Load(criteria exp.Expression, target data.Object, _ ...option.Option) error {

	if !c.found {
		return derp.NotFound("test", "not found")
	}

	if !matchesUser(criteria, c.record) {
		return derp.NotFound("test", "not found")
	}

	user, ok := target.(*model.User)

	if !ok {
		return derp.Internal("test", "unexpected target type")
	}

	*user = c.record
	return nil
}

// Save implements the data.Collection interface. Unused by these tests.
func (c *userCollection) Save(data.Object, string) error { return derp.Internal("test", "unused") }

// Delete implements the data.Collection interface. Unused by these tests.
func (c *userCollection) Delete(data.Object, string) error { return derp.Internal("test", "unused") }

// HardDelete implements the data.Collection interface. Unused by these tests.
func (c *userCollection) HardDelete(exp.Expression) error { return derp.Internal("test", "unused") }

// matchesUser reports whether the stored User satisfies a criteria on username/deleteDate,
// which is exactly what LoadByUsername (wrapped in notDeleted) builds.
func matchesUser(criteria exp.Expression, record model.User) bool {

	return criteria.Match(func(predicate exp.Predicate) bool {

		switch predicate.Field {

		case "username":
			value, ok := predicate.Value.(string)
			return ok && predicate.Operator == exp.OperatorEqual && record.Username == value

		case "deleteDate":
			value, ok := predicate.Value.(int)
			return ok && predicate.Operator == exp.OperatorEqual && record.DeleteDate == int64(value)

		default:
			return false
		}
	})
}

// userSession hands out a single shared userCollection.
type userSession struct {
	collection *userCollection
}

// Collection implements the data.Session interface, returning this stub's single collection
func (s userSession) Collection(string) data.Collection { return s.collection }

// Context implements the interface, returning a background context
func (s userSession) Context() context.Context { return context.Background() }

// Close implements the interface. The stub holds no resources to release.
func (s userSession) Close() {}

// newWebFingerTestService returns a User service and a session holding the provided User.
func newWebFingerTestService(user model.User) (*User, userSession) {
	service := &User{host: "https://example.com"}
	return service, userSession{collection: &userCollection{record: user, found: true}}
}

/******************************************
 * Tests
 ******************************************/

// TestUser_WebFinger_Public confirms that a public profile resolves through WebFinger.
func TestUser_WebFinger_Public(t *testing.T) {

	user := model.NewUser()
	user.UserID = primitive.NewObjectID()
	user.Username = "qapub"
	user.IsPublic = true

	service, session := newWebFingerTestService(user)

	resource, err := service.WebFinger(session, "qapub")

	require.Nil(t, err)
	require.Equal(t, "acct:qapub@example.com", resource.Subject)
}

// TestUser_WebFinger_NonPublic is the regression test for the reported leak: a profile marked
// "Hidden from Public Servers" (isPublic=false) must NOT resolve through WebFinger. Discovery
// is unauthenticated, so there is no requester to exempt -- the answer is a flat NotFound.
func TestUser_WebFinger_NonPublic(t *testing.T) {

	user := model.NewUser()
	user.UserID = primitive.NewObjectID()
	user.Username = "qapwd"
	user.IsPublic = false

	service, session := newWebFingerTestService(user)

	_, err := service.WebFinger(session, "qapwd")

	require.NotNil(t, err)
	require.True(t, derp.IsNotFound(err))
}
