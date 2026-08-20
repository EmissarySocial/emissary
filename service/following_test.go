package service

import (
	"cmp"
	"context"
	"slices"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// These tests use a hand-built data.Collection fake instead of benpate/data-mock, which cannot
// match the criteria that the Folder service builds. They cover setFolder directly, because
// Following.Save fans out to the NewsFeed, User, and Outbox services (plus an SSE channel) while
// the Folder ownership rule lives entirely in setFolder.

/******************************************
 * In-Memory Fakes
 ******************************************/

// folderCollection is an in-memory data.Collection that holds model.Folder records.
type folderCollection struct {
	records []model.Folder
	err     error // When present, every read fails with this error
}

// Context implements the interface, returning a background context
func (c *folderCollection) Context() context.Context { return context.Background() }

// Count implements the data.Collection interface. Unused by these tests.
func (c *folderCollection) Count(exp.Expression, ...option.Option) (int64, error) {
	return 0, derp.Internal("test", "unused")
}

// Query appends every matching Folder to the target slice, in the order that MongoDB would return them
func (c *folderCollection) Query(target any, criteria exp.Expression, options ...option.Option) error {

	if c.err != nil {
		return c.err
	}

	result, ok := target.(*[]model.Folder)

	if !ok {
		return derp.Internal("test", "unexpected target type")
	}

	// Collect the matching records
	for _, record := range c.records {
		if matchesFolder(criteria, record) {
			*result = append(*result, record)
		}
	}

	// Apply an ascending "rank" sort, which is the only option that the Folder service uses
	for _, queryOption := range options {

		sortOption, ok := queryOption.(option.SortOption)

		if !ok {
			continue
		}

		if sortOption.FieldName != "rank" {
			continue
		}

		if sortOption.Direction != option.SortDirectionAscending {
			continue
		}

		slices.SortFunc(*result, func(a model.Folder, b model.Folder) int {
			return cmp.Compare(a.Rank, b.Rank)
		})
	}

	return nil
}

// Iterator implements the data.Collection interface. Unused by these tests.
func (c *folderCollection) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.Internal("test", "unused")
}

// Load copies the first matching Folder into the target
func (c *folderCollection) Load(criteria exp.Expression, target data.Object, _ ...option.Option) error {

	if c.err != nil {
		return c.err
	}

	for _, record := range c.records {

		if !matchesFolder(criteria, record) {
			continue
		}

		folder, ok := target.(*model.Folder)

		if !ok {
			return derp.Internal("test", "unexpected target type")
		}

		*folder = record
		return nil
	}

	return derp.NotFound("test", "not found")
}

// Save implements the data.Collection interface. Unused by these tests.
func (c *folderCollection) Save(data.Object, string) error { return derp.Internal("test", "unused") }

// Delete implements the data.Collection interface. Unused by these tests.
func (c *folderCollection) Delete(data.Object, string) error { return derp.Internal("test", "unused") }

// HardDelete implements the data.Collection interface. Unused by these tests.
func (c *folderCollection) HardDelete(exp.Expression) error { return derp.Internal("test", "unused") }

// matchesFolder reports whether a Folder satisfies a criteria on _id/userId/label/deleteDate.
// "_id" also honors "!=", which Folder.LabelExists uses to exclude the Folder being edited.
func matchesFolder(criteria exp.Expression, record model.Folder) bool {

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
				return record.FolderID == value

			case exp.OperatorNotEqual:
				return record.FolderID != value
			}

			return false

		case "userId":
			value, ok := predicate.Value.(primitive.ObjectID)
			return ok && predicate.Operator == exp.OperatorEqual && record.UserID == value

		case "label":
			value, ok := predicate.Value.(string)
			return ok && predicate.Operator == exp.OperatorEqual && record.Label == value

		case "deleteDate":
			value, ok := predicate.Value.(int)
			return ok && predicate.Operator == exp.OperatorEqual && record.DeleteDate == int64(value)

		default:
			return false
		}
	})
}

// folderSession hands out a single shared folderCollection.
type folderSession struct {
	collection *folderCollection
}

// Collection implements the data.Session interface, returning this stub's single collection
func (s folderSession) Collection(string) data.Collection { return s.collection }

// Context implements the interface, returning a background context
func (s folderSession) Context() context.Context { return context.Background() }

// Close implements the interface. The stub holds no resources to release.
func (s folderSession) Close() {}

// newFollowingService returns a Following service backed by an in-memory set of Folders
func newFollowingService(folders ...model.Folder) (*Following, folderSession) {

	folderService := NewFolder()

	service := NewFollowing()
	service.folderService = &folderService

	return &service, folderSession{collection: &folderCollection{records: folders}}
}

// newTestFolder returns a Folder owned by the provided User
func newTestFolder(userID primitive.ObjectID, label string, rank int) model.Folder {

	folder := model.NewFolder()
	folder.UserID = userID
	folder.Label = label
	folder.Rank = rank

	return folder
}

/******************************************
 * setFolder: Accepted Values
 ******************************************/

// A Folder that belongs to the User is accepted, and its label is cached on the Following.
func TestFollowing_setFolder_Owned(t *testing.T) {

	userID := primitive.NewObjectID()
	folder := newTestFolder(userID, "Personal", 1)
	service, session := newFollowingService(folder)

	following := model.NewFollowing()
	following.UserID = userID
	following.FolderID.Set(folder.FolderID)

	require.Nil(t, service.setFolder(session, &following))
	require.Equal(t, folder.FolderID, following.FolderID.Value())
	require.Equal(t, "Personal", following.Folder)
}

// An empty FolderID falls back to the User's first Folder, which is the one with the lowest rank.
func TestFollowing_setFolder_EmptyUsesFirstFolder(t *testing.T) {

	userID := primitive.NewObjectID()
	second := newTestFolder(userID, "Second", 2)
	first := newTestFolder(userID, "First", 1)

	// Inserted out of rank order, so that insertion order cannot pass this test by accident
	service, session := newFollowingService(second, first)

	following := model.NewFollowing()
	following.UserID = userID

	require.Nil(t, service.setFolder(session, &following))
	require.Equal(t, first.FolderID, following.FolderID.Value())
	require.Equal(t, "First", following.Folder)

	// The repaired FolderID counts as a change, so Save moves existing inbox items into it
	require.True(t, following.FolderID.IsChanged())
}

// Another User's Folders are invisible, even when the fallback picks a default.
func TestFollowing_setFolder_EmptyIgnoresOtherUsersFolders(t *testing.T) {

	userID := primitive.NewObjectID()
	otherUserID := primitive.NewObjectID()

	theirs := newTestFolder(otherUserID, "Theirs", 1)
	mine := newTestFolder(userID, "Mine", 2)

	service, session := newFollowingService(theirs, mine)

	following := model.NewFollowing()
	following.UserID = userID

	require.Nil(t, service.setFolder(session, &following))
	require.Equal(t, mine.FolderID, following.FolderID.Value())
	require.Equal(t, "Mine", following.Folder)
}

/******************************************
 * setFolder: Rejected Values
 ******************************************/

// A Folder that belongs to a different User is rejected, and nothing is written to the Following.
func TestFollowing_setFolder_OtherUsersFolder(t *testing.T) {

	userID := primitive.NewObjectID()
	otherUserID := primitive.NewObjectID()

	mine := newTestFolder(userID, "Mine", 1)
	theirs := newTestFolder(otherUserID, "Theirs", 1)

	service, session := newFollowingService(mine, theirs)

	following := model.NewFollowing()
	following.UserID = userID
	following.FolderID.Set(theirs.FolderID)

	err := service.setFolder(session, &following)

	require.NotNil(t, err)
	require.True(t, derp.IsBadRequest(err))
	require.Empty(t, following.Folder)
}

// A FolderID that exists nowhere is rejected.
func TestFollowing_setFolder_NonexistentFolder(t *testing.T) {

	userID := primitive.NewObjectID()
	service, session := newFollowingService(newTestFolder(userID, "Personal", 1))

	following := model.NewFollowing()
	following.UserID = userID
	following.FolderID.Set(primitive.NewObjectID())

	err := service.setFolder(session, &following)

	require.NotNil(t, err)
	require.True(t, derp.IsBadRequest(err))
	require.Empty(t, following.Folder)
}

// A soft-deleted Folder is not a valid target.
func TestFollowing_setFolder_DeletedFolder(t *testing.T) {

	userID := primitive.NewObjectID()
	folder := newTestFolder(userID, "Deleted", 1)
	folder.DeleteDate = 1000

	service, session := newFollowingService(folder)

	following := model.NewFollowing()
	following.UserID = userID
	following.FolderID.Set(folder.FolderID)

	err := service.setFolder(session, &following)

	require.NotNil(t, err)
	require.True(t, derp.IsBadRequest(err))
	require.Empty(t, following.Folder)
}

// A User with no Folders has nothing to fall back to.
func TestFollowing_setFolder_EmptyWithNoFolders(t *testing.T) {

	service, session := newFollowingService()

	following := model.NewFollowing()
	following.UserID = primitive.NewObjectID()

	err := service.setFolder(session, &following)

	require.NotNil(t, err)
	require.True(t, derp.IsBadRequest(err))
	require.True(t, following.FolderID.Value().IsZero())
}

/******************************************
 * setFolder: Database Failures
 ******************************************/

// A database failure while loading a named Folder is a server error, NOT a rejection of the value.
func TestFollowing_setFolder_LoadFailureIsNotBadRequest(t *testing.T) {

	userID := primitive.NewObjectID()
	folder := newTestFolder(userID, "Personal", 1)
	service, session := newFollowingService(folder)
	session.collection.err = derp.Internal("test", "database unavailable")

	following := model.NewFollowing()
	following.UserID = userID
	following.FolderID.Set(folder.FolderID)

	err := service.setFolder(session, &following)

	require.NotNil(t, err)
	require.False(t, derp.IsBadRequest(err))
	require.True(t, derp.IsServerError(err))
}

// A database failure while listing the User's Folders is a server error, NOT a rejection.
func TestFollowing_setFolder_QueryFailureIsNotBadRequest(t *testing.T) {

	service, session := newFollowingService()
	session.collection.err = derp.Internal("test", "database unavailable")

	following := model.NewFollowing()
	following.UserID = primitive.NewObjectID()

	err := service.setFolder(session, &following)

	require.NotNil(t, err)
	require.False(t, derp.IsBadRequest(err))
	require.True(t, derp.IsServerError(err))
}
