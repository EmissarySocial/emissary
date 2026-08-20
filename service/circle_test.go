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

// These tests cover the Circle name-collision rule that backs the /.validate/circle/name
// endpoint and Circle.Save. They use a hand-built data.Collection fake, mirroring the
// Folder fakes in following_test.go, because benpate/data-mock cannot match the criteria
// that the Circle service builds.

/******************************************
 * In-Memory Fakes
 ******************************************/

// circleCollection is an in-memory data.Collection that holds model.Circle records.
type circleCollection struct {
	records []model.Circle
	err     error // When present, every read fails with this error
}

// Context implements the interface, returning a background context
func (c *circleCollection) Context() context.Context { return context.Background() }

// Count implements the data.Collection interface. Unused by these tests.
func (c *circleCollection) Count(exp.Expression, ...option.Option) (int64, error) {
	return 0, derp.Internal("test", "unused")
}

// Query implements the data.Collection interface. Unused by these tests.
func (c *circleCollection) Query(any, exp.Expression, ...option.Option) error {
	return derp.Internal("test", "unused")
}

// Iterator implements the data.Collection interface. Unused by these tests.
func (c *circleCollection) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.Internal("test", "unused")
}

// Load copies the first matching Circle into the target
func (c *circleCollection) Load(criteria exp.Expression, target data.Object, _ ...option.Option) error {

	if c.err != nil {
		return c.err
	}

	for _, record := range c.records {

		if !matchesCircle(criteria, record) {
			continue
		}

		circle, ok := target.(*model.Circle)

		if !ok {
			return derp.Internal("test", "unexpected target type")
		}

		*circle = record
		return nil
	}

	return derp.NotFound("test", "not found")
}

// Save implements the data.Collection interface. Unused by these tests.
func (c *circleCollection) Save(data.Object, string) error { return derp.Internal("test", "unused") }

// Delete implements the data.Collection interface. Unused by these tests.
func (c *circleCollection) Delete(data.Object, string) error { return derp.Internal("test", "unused") }

// HardDelete implements the data.Collection interface. Unused by these tests.
func (c *circleCollection) HardDelete(exp.Expression) error { return derp.Internal("test", "unused") }

// matchesCircle reports whether a Circle satisfies a criteria on _id/userId/name/deleteDate.
// "_id" also honors "!=", which Circle.NameExists uses to exclude the Circle being edited.
func matchesCircle(criteria exp.Expression, record model.Circle) bool {

	// Any unsupported field or operator conservatively counts as "no match".
	return criteria.Match(func(predicate exp.Predicate) bool {

		switch predicate.Field {

		case "_id":
			value, ok := predicate.Value.(primitive.ObjectID)

			if !ok {
				return false
			}

			switch predicate.Operator {

			case exp.OperatorEqual:
				return record.CircleID == value

			case exp.OperatorNotEqual:
				return record.CircleID != value
			}

			return false

		case "userId":
			value, ok := predicate.Value.(primitive.ObjectID)
			return ok && predicate.Operator == exp.OperatorEqual && record.UserID == value

		case "name":
			value, ok := predicate.Value.(string)
			return ok && predicate.Operator == exp.OperatorEqual && record.Name == value

		case "deleteDate":
			value, ok := predicate.Value.(int)
			return ok && predicate.Operator == exp.OperatorEqual && record.DeleteDate == int64(value)

		default:
			return false
		}
	})
}

// circleSession hands out a single shared circleCollection.
type circleSession struct {
	collection *circleCollection
}

// Collection implements the data.Session interface, returning this stub's single collection
func (s circleSession) Collection(string) data.Collection { return s.collection }

// Context implements the interface, returning a background context
func (s circleSession) Context() context.Context { return context.Background() }

// Close implements the interface. The stub holds no resources to release.
func (s circleSession) Close() {}

// newCircleService returns a Circle service backed by an in-memory set of Circles
func newCircleService(circles ...model.Circle) (*Circle, circleSession) {

	service := NewCircle()

	return &service, circleSession{collection: &circleCollection{records: circles}}
}

// newTestCircle returns a Circle owned by the provided User
func newTestCircle(userID primitive.ObjectID, name string) model.Circle {

	circle := model.NewCircle()
	circle.UserID = userID
	circle.Name = name

	return circle
}

/******************************************
 * ValidateName
 ******************************************/

// A name that nobody is using is available.
func TestCircle_ValidateName_Unused(t *testing.T) {

	userID := primitive.NewObjectID()
	service, session := newCircleService(newTestCircle(userID, "Friends"))

	require.Nil(t, service.ValidateName(session, userID, primitive.NewObjectID(), "Family"))
}

// A name already used by another of this User's Circles is rejected.
func TestCircle_ValidateName_Duplicate(t *testing.T) {

	userID := primitive.NewObjectID()
	service, session := newCircleService(newTestCircle(userID, "Friends"))

	err := service.ValidateName(session, userID, primitive.NewObjectID(), "Friends")

	require.NotNil(t, err)
	require.True(t, derp.IsBadRequest(err))
}

// A Circle may keep its own name when edited -- it does not collide with itself.
func TestCircle_ValidateName_ExcludesItself(t *testing.T) {

	userID := primitive.NewObjectID()
	existing := newTestCircle(userID, "Friends")
	service, session := newCircleService(existing)

	require.Nil(t, service.ValidateName(session, userID, existing.CircleID, "Friends"))
}

// Another User's identical Circle name is not a collision.
func TestCircle_ValidateName_OtherUserIsNotACollision(t *testing.T) {

	userID := primitive.NewObjectID()
	otherUserID := primitive.NewObjectID()
	service, session := newCircleService(newTestCircle(otherUserID, "Friends"))

	require.Nil(t, service.ValidateName(session, userID, primitive.NewObjectID(), "Friends"))
}

// A soft-deleted Circle does not reserve its name.
func TestCircle_ValidateName_DeletedIsNotACollision(t *testing.T) {

	userID := primitive.NewObjectID()
	deleted := newTestCircle(userID, "Friends")
	deleted.DeleteDate = 1000
	service, session := newCircleService(deleted)

	require.Nil(t, service.ValidateName(session, userID, primitive.NewObjectID(), "Friends"))
}

// An empty name is rejected.
func TestCircle_ValidateName_Empty(t *testing.T) {

	service, session := newCircleService()

	err := service.ValidateName(session, primitive.NewObjectID(), primitive.NewObjectID(), "")

	require.NotNil(t, err)
	require.True(t, derp.IsBadRequest(err))
}

// Names are compared exactly: case and whitespace make a distinct Circle.
// Pins current behavior -- tighten deliberately if that is ever wrong.
func TestCircle_ValidateName_ComparisonIsExact(t *testing.T) {

	userID := primitive.NewObjectID()
	service, session := newCircleService(newTestCircle(userID, "Friends"))

	require.Nil(t, service.ValidateName(session, userID, primitive.NewObjectID(), "friends"))
	require.Nil(t, service.ValidateName(session, userID, primitive.NewObjectID(), "Friends "))
}
