package service

import (
	"context"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// These tests use hand-built data.Collection fakes, not benpate/data-mock: the
// mock matches on the raw bson tag string, so it can't match `parentId,omitempty`
// / `type,omitempty` (it never strips the ",omitempty"). The real concurrency
// guarantee lives in the unique index (queries/sync/collection.go); the fakes pin
// the service's reaction to it — the load/create decision and duplicate-key retry.

// memoryCollection is an in-memory data.Collection for testing.
// It matches on the fields LoadOrCreateByParent queries: parentId, type, deleteDate.
type memoryCollection struct {
	records []*model.Collection
}

func (c *memoryCollection) Context() context.Context { return context.Background() }

func (c *memoryCollection) Count(criteria exp.Expression, _ ...option.Option) (int64, error) {
	var count int64
	for _, record := range c.records {
		if matchesCollection(criteria, record) {
			count++
		}
	}
	return count, nil
}

func (c *memoryCollection) Query(any, exp.Expression, ...option.Option) error {
	return derp.NotFound("test", "unused")
}

func (c *memoryCollection) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.NotFound("test", "unused")
}

func (c *memoryCollection) Load(criteria exp.Expression, target data.Object, _ ...option.Option) error {
	for _, record := range c.records {
		if matchesCollection(criteria, record) {
			if collection, ok := target.(*model.Collection); ok {
				*collection = *record
				return nil
			}
			return derp.Internal("test", "unexpected target type")
		}
	}
	return derp.NotFound("test", "not found")
}

func (c *memoryCollection) Save(object data.Object, _ string) error {
	collection, ok := object.(*model.Collection)
	if !ok {
		return derp.Internal("test", "unexpected object type")
	}

	// Update in place if the ID already exists; otherwise insert a copy.
	for index, record := range c.records {
		if record.CollectionID == collection.CollectionID {
			saved := *collection
			c.records[index] = &saved
			return nil
		}
	}

	saved := *collection
	c.records = append(c.records, &saved)
	return nil
}

func (c *memoryCollection) Delete(data.Object, string) error { return nil }
func (c *memoryCollection) HardDelete(exp.Expression) error  { return nil }

// matchesCollection reports whether a record satisfies an equality criteria on parentId/type/deleteDate.
func matchesCollection(criteria exp.Expression, record *model.Collection) bool {

	// Any unsupported field or operator conservatively counts as "no match".
	return criteria.Match(func(predicate exp.Predicate) bool {
		if predicate.Operator != exp.OperatorEqual {
			return false
		}
		switch predicate.Field {
		case "parentId":
			value, ok := predicate.Value.(primitive.ObjectID)
			return ok && record.ParentID == value
		case "type":
			value, ok := predicate.Value.(string)
			return ok && record.Type == value
		case "deleteDate":
			value, ok := predicate.Value.(int)
			return ok && record.DeleteDate == int64(value)
		default:
			return false
		}
	})
}

// memorySession hands out a single shared memoryCollection.
type memorySession struct {
	collection *memoryCollection
}

func (s memorySession) Collection(string) data.Collection { return s.collection }
func (s memorySession) Context() context.Context          { return context.Background() }
func (s memorySession) Close()                            {}

func newMemoryService() (*Collection, memorySession) {
	service := NewCollection()
	service.collectionItemService = &CollectionItem{}
	return &service, memorySession{collection: &memoryCollection{}}
}

/******************************************
 * LoadOrCreateByParent — happy paths
 ******************************************/

// When no Collection exists, LoadOrCreateByParent creates one with owner, parent, and type set.
func TestCollection_LoadOrCreateByParent_Creates(t *testing.T) {

	service, session := newMemoryService()

	ownerID := primitive.NewObjectID()
	parentID := primitive.NewObjectID()

	read := sliceof.String{"https://reader.test/@r"}
	write := sliceof.String{"https://writer.test/@w"}

	collection := model.NewCollection()
	err := service.LoadOrCreateByParent(session, ownerID, parentID, model.CollectionTypeReplies, read, write, &collection)

	require.Nil(t, err)
	require.False(t, collection.CollectionID.IsZero())
	require.Equal(t, ownerID, collection.UserID)
	require.Equal(t, parentID, collection.ParentID)
	require.Equal(t, model.CollectionTypeReplies, collection.Type)

	// The read/write permissions passed in are applied on create.
	require.Equal(t, read, collection.Read)
	require.Equal(t, write, collection.Write)

	// The record must actually be persisted (a subsequent load finds it).
	loaded := model.NewCollection()
	require.Nil(t, service.LoadByParentAndType(session, parentID, model.CollectionTypeReplies, &loaded))
	require.Equal(t, collection.CollectionID, loaded.CollectionID)
}

// The read/write permissions are applied ONLY on create; an existing collection keeps its own.
func TestCollection_LoadOrCreateByParent_KeepsExistingPermissions(t *testing.T) {

	service, session := newMemoryService()

	ownerID := primitive.NewObjectID()
	parentID := primitive.NewObjectID()

	original := sliceof.String{"https://original.test/@o"}
	first := model.NewCollection()
	require.Nil(t, service.LoadOrCreateByParent(session, ownerID, parentID, model.CollectionTypeReplies, original, original, &first))

	// A second call with DIFFERENT permissions must not overwrite the existing ones.
	different := sliceof.String{"https://different.test/@d"}
	second := model.NewCollection()
	require.Nil(t, service.LoadOrCreateByParent(session, ownerID, parentID, model.CollectionTypeReplies, different, different, &second))

	require.Equal(t, original, second.Read)
	require.Equal(t, original, second.Write)
}

// When a Collection already exists, LoadOrCreateByParent returns it and creates no duplicate.
func TestCollection_LoadOrCreateByParent_LoadsExisting(t *testing.T) {

	service, session := newMemoryService()

	ownerID := primitive.NewObjectID()
	parentID := primitive.NewObjectID()

	// First call creates the collection.
	first := model.NewCollection()
	require.Nil(t, service.LoadOrCreateByParent(session, ownerID, parentID, model.CollectionTypeContext, nil, nil, &first))

	// Second call, same (parentID, type), must return the SAME collection.
	second := model.NewCollection()
	require.Nil(t, service.LoadOrCreateByParent(session, ownerID, parentID, model.CollectionTypeContext, nil, nil, &second))

	require.Equal(t, first.CollectionID, second.CollectionID)

	// Exactly one live collection exists for this (parentID, type).
	count, err := service.Count(session, exp.Equal("parentId", parentID).AndEqual("type", model.CollectionTypeContext))
	require.Nil(t, err)
	require.Equal(t, int64(1), count)
}

// A different (parentID, type) yields a distinct collection.
func TestCollection_LoadOrCreateByParent_DistinctByType(t *testing.T) {

	service, session := newMemoryService()

	ownerID := primitive.NewObjectID()
	parentID := primitive.NewObjectID()

	replies := model.NewCollection()
	require.Nil(t, service.LoadOrCreateByParent(session, ownerID, parentID, model.CollectionTypeReplies, nil, nil, &replies))

	context := model.NewCollection()
	require.Nil(t, service.LoadOrCreateByParent(session, ownerID, parentID, model.CollectionTypeContext, nil, nil, &context))

	require.NotEqual(t, replies.CollectionID, context.CollectionID)
}

/******************************************
 * LoadOrCreateByParent — duplicate-key retry
 ******************************************/

// duplicateKeyError returns an error shaped like MongoDB's unique-index violation.
func duplicateKeyError() error {

	// Code 11000 is what mongo.IsDuplicateKeyError looks for.
	return mongo.WriteException{
		WriteErrors: mongo.WriteErrors{
			{Code: 11000, Message: "E11000 duplicate key error"},
		},
	}
}

// raceCollection is a data.Collection that simulates losing a creation race.
//
// It models the sequence LoadOrCreateByParent must survive: (1) the initial Load
// misses, (2) the optimistic Save is rejected with a duplicate-key error because a
// competing writer committed first, (3) the re-Load finds that writer's record.
// saveErr is returned by every Save; winnerOnConflict becomes visible to Load only
// AFTER a Save is attempted, which is what makes step (1) miss and step (3) hit.
type raceCollection struct {
	saveErr          error
	winnerOnConflict *model.Collection
	saveCalls        int
	saved            *model.Collection // captured on a successful Save
}

func (c *raceCollection) Context() context.Context { return context.Background() }

func (c *raceCollection) Count(exp.Expression, ...option.Option) (int64, error) {
	return 0, nil
}

func (c *raceCollection) Query(any, exp.Expression, ...option.Option) error {
	return derp.NotFound("test", "unused")
}

func (c *raceCollection) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.NotFound("test", "unused")
}

// Load returns the visible record, or NotFound before anything is visible.
func (c *raceCollection) Load(_ exp.Expression, target data.Object, _ ...option.Option) error {

	var visible *model.Collection

	switch {
	case c.saveCalls > 0 && c.winnerOnConflict != nil:
		visible = c.winnerOnConflict
	case c.saved != nil:
		visible = c.saved
	}

	if visible == nil {
		return derp.NotFound("test", "no collection yet")
	}

	if collection, ok := target.(*model.Collection); ok {
		*collection = *visible
		return nil
	}

	return derp.Internal("test", "unexpected target type")
}

func (c *raceCollection) Save(object data.Object, _ string) error {

	c.saveCalls++

	if c.saveErr != nil {
		return c.saveErr
	}

	if collection, ok := object.(*model.Collection); ok {
		saved := *collection
		c.saved = &saved
	}

	return nil
}

func (c *raceCollection) Delete(data.Object, string) error { return nil }

func (c *raceCollection) HardDelete(exp.Expression) error { return nil }

// raceSession hands out a single shared raceCollection.
type raceSession struct {
	collection *raceCollection
}

func (s raceSession) Collection(string) data.Collection { return s.collection }
func (s raceSession) Context() context.Context          { return context.Background() }
func (s raceSession) Close()                            {}

// When the insert loses the race, the service detects the duplicate key and returns the winner.
func TestCollection_LoadOrCreateByParent_DuplicateKeyReloadsWinner(t *testing.T) {

	ownerID := primitive.NewObjectID()
	parentID := primitive.NewObjectID()

	// The record a competing writer committed first. It becomes visible to Load
	// only after our Save is attempted (see raceCollection.Load).
	winner := model.NewCollection()
	winner.UserID = ownerID
	winner.ParentID = parentID
	winner.Type = model.CollectionTypeReplies

	race := &raceCollection{
		saveErr:          duplicateKeyError(),
		winnerOnConflict: &winner,
	}
	session := raceSession{collection: race}

	service := NewCollection()
	service.collectionItemService = &CollectionItem{}

	collection := model.NewCollection()
	err := service.LoadOrCreateByParent(session, ownerID, parentID, model.CollectionTypeReplies, nil, nil, &collection)

	require.Nil(t, err)
	require.Equal(t, winner.CollectionID, collection.CollectionID) // returned the winner
	require.Equal(t, 1, race.saveCalls)                            // exactly one insert attempt
}

// When the duplicate-key re-load ALSO fails, the error propagates (no silent zero-value collection).
func TestCollection_LoadOrCreateByParent_DuplicateKeyReloadFails(t *testing.T) {

	// saveErr triggers the duplicate-key branch, but winnerOnConflict stays nil,
	// so the post-conflict re-Load misses (returns NotFound) and that error must
	// surface.
	race := &raceCollection{
		saveErr:          duplicateKeyError(),
		winnerOnConflict: nil,
	}
	session := raceSession{collection: race}

	service := NewCollection()
	service.collectionItemService = &CollectionItem{}

	collection := model.NewCollection()
	err := service.LoadOrCreateByParent(session, primitive.NewObjectID(), primitive.NewObjectID(), model.CollectionTypeReplies, nil, nil, &collection)

	require.NotNil(t, err)
	require.True(t, derp.IsNotFound(err)) // the re-load's NotFound propagates
}

// A non-duplicate Save error is a real failure and must propagate unchanged.
func TestCollection_LoadOrCreateByParent_SaveErrorPropagates(t *testing.T) {

	race := &raceCollection{
		saveErr: derp.Internal("test", "disk on fire"),
	}
	session := raceSession{collection: race}

	service := NewCollection()
	service.collectionItemService = &CollectionItem{}

	collection := model.NewCollection()
	err := service.LoadOrCreateByParent(session, primitive.NewObjectID(), primitive.NewObjectID(), model.CollectionTypeReplies, nil, nil, &collection)

	require.NotNil(t, err)
	require.False(t, mongo.IsDuplicateKeyError(err))
}

// A non-NotFound load error propagates without the service attempting to create a collection.
func TestCollection_LoadOrCreateByParent_LoadErrorPropagates(t *testing.T) {

	hardLoad := &hardLoadCollection{loadErr: derp.Internal("test", "index corrupt")}
	session := fakeSession{collection: hardLoad}

	service := NewCollection()
	service.collectionItemService = &CollectionItem{}

	collection := model.NewCollection()
	err := service.LoadOrCreateByParent(session, primitive.NewObjectID(), primitive.NewObjectID(), model.CollectionTypeReplies, nil, nil, &collection)

	require.NotNil(t, err)
	require.Equal(t, 0, hardLoad.saveCalls) // never attempted to create
}

// hardLoadCollection is a data.Collection whose Load always fails with a non-NotFound error.
type hardLoadCollection struct {
	loadErr   error
	saveCalls int
}

func (c *hardLoadCollection) Context() context.Context                              { return context.Background() }
func (c *hardLoadCollection) Count(exp.Expression, ...option.Option) (int64, error) { return 0, nil }
func (c *hardLoadCollection) Query(any, exp.Expression, ...option.Option) error {
	return derp.NotFound("test", "unused")
}
func (c *hardLoadCollection) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.NotFound("test", "unused")
}
func (c *hardLoadCollection) Load(exp.Expression, data.Object, ...option.Option) error {
	return c.loadErr
}
func (c *hardLoadCollection) Save(data.Object, string) error {
	c.saveCalls++
	return nil
}
func (c *hardLoadCollection) Delete(data.Object, string) error { return nil }
func (c *hardLoadCollection) HardDelete(exp.Expression) error  { return nil }

// fakeSession hands out any data.Collection.
type fakeSession struct {
	collection data.Collection
}

func (s fakeSession) Collection(string) data.Collection { return s.collection }
func (s fakeSession) Context() context.Context          { return context.Background() }
func (s fakeSession) Close()                            {}
