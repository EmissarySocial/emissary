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
	"go.mongodb.org/mongo-driver/mongo"
)

// These tests use hand-built data.Collection fakes for the same reason as
// collection_test.go: benpate/data-mock matches on the raw bson tag string and
// can't match the model's `,omitempty` fields. The real concurrency guarantee
// lives in the unique (collectionId, uri) index (queries/sync/collectionItem.go);
// the fakes pin SaveUnique's reaction to it — the merge-or-insert decision and
// the duplicate-key retry.

/******************************************
 * itemStore — an in-memory data.Collection that matches CollectionItems on the
 * fields SaveUnique queries: collectionId, uri, deleteDate.
 ******************************************/

type itemStore struct {
	records []*model.CollectionItem
}

func (c *itemStore) Context() context.Context { return context.Background() }

func (c *itemStore) Count(criteria exp.Expression, _ ...option.Option) (int64, error) {
	var count int64
	for _, record := range c.records {
		if matchesItem(criteria, record) {
			count++
		}
	}
	return count, nil
}

func (c *itemStore) Query(any, exp.Expression, ...option.Option) error {
	return derp.NotFound("test", "unused")
}

func (c *itemStore) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.NotFound("test", "unused")
}

func (c *itemStore) Load(criteria exp.Expression, target data.Object, _ ...option.Option) error {
	for _, record := range c.records {
		if matchesItem(criteria, record) {
			if item, ok := target.(*model.CollectionItem); ok {
				*item = *record
				return nil
			}
			return derp.Internal("test", "unexpected target type")
		}
	}
	return derp.NotFound("test", "not found")
}

func (c *itemStore) Save(object data.Object, _ string) error {
	item, ok := object.(*model.CollectionItem)
	if !ok {
		return derp.Internal("test", "unexpected object type")
	}

	// Update in place when this item has an identity that already exists.
	if !item.CollectionItemID.IsZero() {
		for index, record := range c.records {
			if record.CollectionItemID == item.CollectionItemID {
				saved := *item
				c.records[index] = &saved
				return nil
			}
		}
	}

	// Otherwise insert. Stamp an ID so a subsequent Load can find it.
	saved := *item
	if saved.CollectionItemID.IsZero() {
		saved.CollectionItemID = primitive.NewObjectID()
	}
	saved.CreateDate = 1 // mark as persisted (IsNew keys off CreateDate)
	c.records = append(c.records, &saved)
	return nil
}

func (c *itemStore) Delete(data.Object, string) error { return nil }
func (c *itemStore) HardDelete(exp.Expression) error  { return nil }

// matchesItem reports whether a record satisfies an equality criteria on collectionId/uri/deleteDate.
func matchesItem(criteria exp.Expression, record *model.CollectionItem) bool {

	// Any unsupported field or operator conservatively counts as "no match".
	return criteria.Match(func(predicate exp.Predicate) bool {
		if predicate.Operator != exp.OperatorEqual {
			return false
		}
		switch predicate.Field {
		case "collectionId":
			value, ok := predicate.Value.(primitive.ObjectID)
			return ok && record.CollectionID == value
		case "uri":
			value, ok := predicate.Value.(string)
			return ok && record.URI == value
		case "inReplyTo":
			value, ok := predicate.Value.(string)
			return ok && record.InReplyTo == value
		case "deleteDate":
			value, ok := predicate.Value.(int)
			return ok && record.DeleteDate == int64(value)
		default:
			return false
		}
	})
}

type itemSession struct {
	store data.Collection
}

func (s itemSession) Collection(string) data.Collection { return s.store }
func (s itemSession) Context() context.Context          { return context.Background() }
func (s itemSession) Close()                            {}

func newItemService(store data.Collection) (*CollectionItem, itemSession) {
	service := NewCollectionItem()
	return &service, itemSession{store: store}
}

func newItem(collectionID primitive.ObjectID, uri string) *model.CollectionItem {
	item := model.NewCollectionItem()
	item.UserID = primitive.NewObjectID()
	item.CollectionID = collectionID
	item.URI = uri
	return &item
}

// itemReplyingTo returns a CollectionItem stored under a given inReplyTo target.
func itemReplyingTo(inReplyTo string) *model.CollectionItem {
	item := model.NewCollectionItem()
	item.CollectionItemID = primitive.NewObjectID()
	item.URI = "https://example.test/reply/" + primitive.NewObjectID().Hex()
	item.InReplyTo = inReplyTo
	return &item
}

/******************************************
 * CountByInReplyTo
 ******************************************/

// CountByInReplyTo counts only the items that reply to the given URI.
func TestCollectionItem_CountByInReplyTo(t *testing.T) {

	const target = "https://example.test/parent"
	const other = "https://example.test/somewhere-else"

	store := &itemStore{
		records: []*model.CollectionItem{
			itemReplyingTo(target),
			itemReplyingTo(target),
			itemReplyingTo(other), // must not be counted
		},
	}
	service, session := newItemService(store)

	count, err := service.CountByInReplyTo(session, target)
	require.Nil(t, err)
	require.Equal(t, int64(2), count)
}

// CountByInReplyTo returns zero when nothing replies to the URI.
func TestCollectionItem_CountByInReplyTo_None(t *testing.T) {

	store := &itemStore{}
	service, session := newItemService(store)

	count, err := service.CountByInReplyTo(session, "https://example.test/lonely")
	require.Nil(t, err)
	require.Zero(t, count)
}

/******************************************
 * SaveUnique — merge/insert behavior
 ******************************************/

// A first SaveUnique inserts the item.
func TestCollectionItem_SaveUnique_Inserts(t *testing.T) {

	store := &itemStore{}
	service, session := newItemService(store)

	item := newItem(primitive.NewObjectID(), "https://example.test/1")
	require.Nil(t, service.SaveUnique(session, item, "insert"))

	require.Len(t, store.records, 1)
}

// A second SaveUnique for the same (collection, uri) updates in place — no duplicate.
func TestCollectionItem_SaveUnique_FoldsDuplicateURI(t *testing.T) {

	store := &itemStore{}
	service, session := newItemService(store)

	collectionID := primitive.NewObjectID()

	first := newItem(collectionID, "https://example.test/dup")
	require.Nil(t, service.SaveUnique(session, first, "first"))

	second := newItem(collectionID, "https://example.test/dup")
	require.Nil(t, service.SaveUnique(session, second, "second"))

	// Still exactly one record, and the second reused the first's identity.
	require.Len(t, store.records, 1)
	require.Equal(t, store.records[0].CollectionItemID, second.CollectionItemID)
}

// The same URI in a DIFFERENT collection is a distinct item.
func TestCollectionItem_SaveUnique_DistinctByCollection(t *testing.T) {

	store := &itemStore{}
	service, session := newItemService(store)

	const uri = "https://example.test/shared"
	require.Nil(t, service.SaveUnique(session, newItem(primitive.NewObjectID(), uri), "a"))
	require.Nil(t, service.SaveUnique(session, newItem(primitive.NewObjectID(), uri), "b"))

	require.Len(t, store.records, 2)
}

/******************************************
 * SaveUnique — duplicate-key retry
 ******************************************/

// itemRaceStore fails the first Save with a duplicate-key error, as the unique
// index would when a competing writer inserts the same (collection, uri) first.
// The competing record becomes visible to Load only AFTER that failed Save.
type itemRaceStore struct {
	winnerOnConflict *model.CollectionItem
	saveErr          error
	saveCalls        int
	saved            []*model.CollectionItem
}

func (c *itemRaceStore) Context() context.Context { return context.Background() }
func (c *itemRaceStore) Count(exp.Expression, ...option.Option) (int64, error) {
	return 0, nil
}
func (c *itemRaceStore) Query(any, exp.Expression, ...option.Option) error {
	return derp.NotFound("test", "unused")
}
func (c *itemRaceStore) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.NotFound("test", "unused")
}

func (c *itemRaceStore) Load(_ exp.Expression, target data.Object, _ ...option.Option) error {
	if c.saveCalls > 0 && c.winnerOnConflict != nil {
		if item, ok := target.(*model.CollectionItem); ok {
			*item = *c.winnerOnConflict
			return nil
		}
	}
	return derp.NotFound("test", "not found")
}

func (c *itemRaceStore) Save(object data.Object, _ string) error {
	c.saveCalls++

	// The FIRST save (the optimistic insert) hits the injected duplicate-key error.
	if c.saveErr != nil && c.saveCalls == 1 {
		return c.saveErr
	}

	if item, ok := object.(*model.CollectionItem); ok {
		saved := *item
		c.saved = append(c.saved, &saved)
	}
	return nil
}

func (c *itemRaceStore) Delete(data.Object, string) error { return nil }
func (c *itemRaceStore) HardDelete(exp.Expression) error  { return nil }

// When the optimistic insert loses the race, SaveUnique merges onto the winner
// and updates in place rather than erroring or double-inserting.
func TestCollectionItem_SaveUnique_DuplicateKeyMergesOntoWinner(t *testing.T) {

	collectionID := primitive.NewObjectID()

	winner := model.NewCollectionItem()
	winner.CollectionItemID = primitive.NewObjectID()
	winner.CollectionID = collectionID
	winner.URI = "https://example.test/raced"
	winner.CreateDate = 100

	race := &itemRaceStore{
		saveErr:          duplicateKeyError(),
		winnerOnConflict: &winner,
	}
	service, session := newItemService(race)

	item := newItem(collectionID, "https://example.test/raced")
	err := service.SaveUnique(session, item, "loser")

	require.Nil(t, err)
	require.Equal(t, winner.CollectionItemID, item.CollectionItemID) // merged onto the winner
	require.Equal(t, 2, race.saveCalls)                              // one failed insert, one successful update
}

// A non-duplicate Save error propagates unchanged.
func TestCollectionItem_SaveUnique_SaveErrorPropagates(t *testing.T) {

	race := &itemRaceStore{
		saveErr: derp.Internal("test", "disk on fire"),
	}
	service, session := newItemService(race)

	item := newItem(primitive.NewObjectID(), "https://example.test/x")
	err := service.SaveUnique(session, item, "boom")

	require.NotNil(t, err)
	require.False(t, mongo.IsDuplicateKeyError(err))
}

// itemHardLoadStore returns a non-NotFound error from every Load.
type itemHardLoadStore struct {
	loadErr   error
	saveCalls int
}

func (c *itemHardLoadStore) Context() context.Context { return context.Background() }
func (c *itemHardLoadStore) Count(exp.Expression, ...option.Option) (int64, error) {
	return 0, nil
}
func (c *itemHardLoadStore) Query(any, exp.Expression, ...option.Option) error {
	return derp.NotFound("test", "unused")
}
func (c *itemHardLoadStore) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.NotFound("test", "unused")
}
func (c *itemHardLoadStore) Load(exp.Expression, data.Object, ...option.Option) error {
	return c.loadErr
}
func (c *itemHardLoadStore) Save(data.Object, string) error {
	c.saveCalls++
	return nil
}
func (c *itemHardLoadStore) Delete(data.Object, string) error { return nil }
func (c *itemHardLoadStore) HardDelete(exp.Expression) error  { return nil }

// A non-NotFound error from the initial existence check propagates, and no Save
// is attempted.
func TestCollectionItem_SaveUnique_MergeLoadErrorPropagates(t *testing.T) {

	store := &itemHardLoadStore{loadErr: derp.Internal("test", "index corrupt")}
	service, session := newItemService(store)

	item := newItem(primitive.NewObjectID(), "https://example.test/y")
	err := service.SaveUnique(session, item, "boom")

	require.NotNil(t, err)
	require.Equal(t, 0, store.saveCalls) // never reached Save
}

// When the insert loses the race but the post-conflict re-merge Load ALSO fails,
// the error propagates rather than silently succeeding.
func TestCollectionItem_SaveUnique_DuplicateKeyRemergeFails(t *testing.T) {

	// dupKeyOnce makes the first Save fail with a duplicate-key error; loadErr
	// (a non-NotFound error) then makes the post-conflict re-merge fail. The
	// initial existence check must miss, so the first Load returns NotFound —
	// but this store returns loadErr on EVERY Load, so we sequence it: NotFound
	// first, hard error after the failed Save.
	store := &itemRemergeStore{}
	service, session := newItemService(store)

	item := newItem(primitive.NewObjectID(), "https://example.test/z")
	err := service.SaveUnique(session, item, "loser")

	require.NotNil(t, err)
	require.False(t, mongo.IsDuplicateKeyError(err)) // the re-merge load error, not the dup-key
}

// When the insert loses the race and the re-merge succeeds but the follow-up
// update Save fails, that error propagates.
func TestCollectionItem_SaveUnique_DuplicateKeyResaveFails(t *testing.T) {

	store := &itemResaveStore{}
	service, session := newItemService(store)

	item := newItem(primitive.NewObjectID(), "https://example.test/w")
	err := service.SaveUnique(session, item, "loser")

	require.NotNil(t, err)
	require.Equal(t, 2, store.saveCalls) // failed insert, then failed update
}

// itemResaveStore: first Load misses, first Save hits duplicate-key, re-merge
// Load succeeds, second Save (the update) fails hard.
type itemResaveStore struct {
	loadCalls int
	saveCalls int
}

func (c *itemResaveStore) Context() context.Context { return context.Background() }
func (c *itemResaveStore) Count(exp.Expression, ...option.Option) (int64, error) {
	return 0, nil
}
func (c *itemResaveStore) Query(any, exp.Expression, ...option.Option) error {
	return derp.NotFound("test", "unused")
}
func (c *itemResaveStore) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.NotFound("test", "unused")
}
func (c *itemResaveStore) Load(_ exp.Expression, target data.Object, _ ...option.Option) error {
	c.loadCalls++
	if c.loadCalls == 1 {
		return derp.NotFound("test", "initial miss")
	}
	// Re-merge succeeds: hand back a winner.
	if item, ok := target.(*model.CollectionItem); ok {
		item.CollectionItemID = primitive.NewObjectID()
		item.CreateDate = 100
	}
	return nil
}
func (c *itemResaveStore) Save(data.Object, string) error {
	c.saveCalls++
	if c.saveCalls == 1 {
		return duplicateKeyError()
	}
	return derp.Internal("test", "update failed")
}
func (c *itemResaveStore) Delete(data.Object, string) error { return nil }
func (c *itemResaveStore) HardDelete(exp.Expression) error  { return nil }

// itemRemergeStore: first Load misses (NotFound), first Save hits duplicate-key,
// second Load (the re-merge) fails hard.
type itemRemergeStore struct {
	loadCalls int
	saveCalls int
}

func (c *itemRemergeStore) Context() context.Context { return context.Background() }
func (c *itemRemergeStore) Count(exp.Expression, ...option.Option) (int64, error) {
	return 0, nil
}
func (c *itemRemergeStore) Query(any, exp.Expression, ...option.Option) error {
	return derp.NotFound("test", "unused")
}
func (c *itemRemergeStore) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.NotFound("test", "unused")
}
func (c *itemRemergeStore) Load(exp.Expression, data.Object, ...option.Option) error {
	c.loadCalls++
	if c.loadCalls == 1 {
		return derp.NotFound("test", "initial miss")
	}
	return derp.Internal("test", "index corrupt on re-merge")
}
func (c *itemRemergeStore) Save(data.Object, string) error {
	c.saveCalls++
	if c.saveCalls == 1 {
		return duplicateKeyError()
	}
	return nil
}
func (c *itemRemergeStore) Delete(data.Object, string) error { return nil }
func (c *itemRemergeStore) HardDelete(exp.Expression) error  { return nil }
