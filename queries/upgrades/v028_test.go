package upgrades

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// newCollectionRecord builds one live Collection row for the dedupe planner, stamped with a createDate
// so the newest-wins rule has something to sort on.
func newCollectionRecord(parentID primitive.ObjectID, collectionType string, createDate int64) collectionRecord {
	return collectionRecord{
		CollectionID:   primitive.NewObjectID(),
		ParentID:       parentID,
		CollectionType: collectionType,
		CreateDate:     createDate,
	}
}

/******************************************
 * planCollectionDedupe
 ******************************************/

// Two live collections of DIFFERENT types under one parent are NOT duplicates -- this is the reported
// failure, where a Context and a Replies collection both read collectionType:null (value stranded in
// the legacy `type` field) and collided only because the rename had not run. After the rename each
// carries its own type, so both survive.
func TestPlanCollectionDedupe_DistinctTypesSurvive(t *testing.T) {

	parentID := primitive.NewObjectID()

	records := []collectionRecord{
		newCollectionRecord(parentID, model.CollectionTypeContext, 100),
		newCollectionRecord(parentID, model.CollectionTypeReplies, 200),
	}

	require.Empty(t, planCollectionDedupe(records))
}

// Two live collections of the SAME type under one parent ARE duplicates: the newest survives, the older
// is deleted.
func TestPlanCollectionDedupe_DuplicatesKeepNewest(t *testing.T) {

	parentID := primitive.NewObjectID()

	older := newCollectionRecord(parentID, model.CollectionTypeLikes, 100)
	newer := newCollectionRecord(parentID, model.CollectionTypeLikes, 200)

	require.Equal(t, []primitive.ObjectID{older.CollectionID}, planCollectionDedupe([]collectionRecord{older, newer}))
}

// The same collectionType under two DIFFERENT parents is not a collision: the unique index spans
// (parentId, collectionType), so both survive.
func TestPlanCollectionDedupe_SameTypeDifferentParentsSurvive(t *testing.T) {

	collectionA := newCollectionRecord(primitive.NewObjectID(), model.CollectionTypeLikes, 100)
	collectionB := newCollectionRecord(primitive.NewObjectID(), model.CollectionTypeLikes, 100)

	require.Empty(t, planCollectionDedupe([]collectionRecord{collectionA, collectionB}))
}

// Rows whose collectionType is still empty (the rename could not find a value to adopt) collide on
// (parentId, "") just like any other repeated key: the newest survives so the index can build.
func TestPlanCollectionDedupe_EmptyTypeDuplicatesCollapse(t *testing.T) {

	parentID := primitive.NewObjectID()

	older := newCollectionRecord(parentID, "", 100)
	newer := newCollectionRecord(parentID, "", 200)

	require.Equal(t, []primitive.ObjectID{older.CollectionID}, planCollectionDedupe([]collectionRecord{older, newer}))
}

// A clean collection with one row per (parent, type) is left entirely alone.
func TestPlanCollectionDedupe_CleanCollectionUntouched(t *testing.T) {

	parentID := primitive.NewObjectID()

	records := []collectionRecord{
		newCollectionRecord(parentID, model.CollectionTypeContext, 100),
		newCollectionRecord(parentID, model.CollectionTypeLikes, 100),
		newCollectionRecord(parentID, model.CollectionTypeShares, 100),
	}

	require.Empty(t, planCollectionDedupe(records))
}

// An empty collection produces an empty plan without panicking.
func TestPlanCollectionDedupe_Empty(t *testing.T) {
	require.Empty(t, planCollectionDedupe([]collectionRecord{}))
}

/******************************************
 * newestCollection
 ******************************************/

// The newest record wins regardless of its position in the group.
func TestNewestCollection(t *testing.T) {

	parentID := primitive.NewObjectID()

	first := newCollectionRecord(parentID, model.CollectionTypeLikes, 300)
	middle := newCollectionRecord(parentID, model.CollectionTypeLikes, 100)
	last := newCollectionRecord(parentID, model.CollectionTypeLikes, 200)

	require.Equal(t, first.CollectionID, newestCollection([]collectionRecord{first, middle, last}).CollectionID)
	require.Equal(t, first.CollectionID, newestCollection([]collectionRecord{middle, last, first}).CollectionID)
	require.Equal(t, first.CollectionID, newestCollection([]collectionRecord{middle, first, last}).CollectionID)
}

// A single-record group returns that record.
func TestNewestCollection_Single(t *testing.T) {

	only := newCollectionRecord(primitive.NewObjectID(), model.CollectionTypeLikes, 100)

	require.Equal(t, only.CollectionID, newestCollection([]collectionRecord{only}).CollectionID)
}
