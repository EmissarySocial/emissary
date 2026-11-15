package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// reorderFixture holds "photo" Attachments on one Stream in a throwaway database, keyed by
// the rank each started with. It runs against the local replica set, because data-mock cannot
// match the journal's deleteDate, which every Attachment read filters on.
type reorderFixture struct {
	service  *Attachment
	session  data.Session
	objectID primitive.ObjectID
	byRank   map[int]primitive.ObjectID
}

// newReorderFixture stores one "photo" Attachment for each distinct rank given, in the order given
func newReorderFixture(t *testing.T, ranks ...int) reorderFixture {
	t.Helper()

	_, session := newReplicaSetSession(t)
	service := NewAttachment()

	fixture := reorderFixture{
		service:  &service,
		session:  session,
		objectID: primitive.NewObjectID(),
		byRank:   map[int]primitive.ObjectID{},
	}

	for _, rank := range ranks {
		fixture.byRank[rank] = fixture.store(t, "Stream", fixture.objectID, "photo", rank)
	}

	return fixture
}

// store writes one Attachment straight to the database and returns its ID
func (fixture reorderFixture) store(t *testing.T, objectType string, objectID primitive.ObjectID, category string, rank int) primitive.ObjectID {
	t.Helper()
	return fixture.storeWithStatus(t, objectType, objectID, category, rank, "")
}

// storeWithStatus writes one Attachment straight to the database, with the given status, and
// returns its ID. A status outside the schema's enum makes the next validated save of it fail.
func (fixture reorderFixture) storeWithStatus(t *testing.T, objectType string, objectID primitive.ObjectID, category string, rank int, status string) primitive.ObjectID {
	t.Helper()

	attachment := model.NewAttachment(objectType, objectID)
	attachment.Category = category
	attachment.Original = "file.jpg"
	attachment.Rank = rank
	attachment.Status = status
	require.NoError(t, fixture.session.Collection("Attachment").Save(&attachment, "seed"))

	return attachment.AttachmentID
}

// ids returns the IDs of the Attachments that started at the given ranks, in that order
func (fixture reorderFixture) ids(originalRanks ...int) []primitive.ObjectID {

	result := make([]primitive.ObjectID, 0, len(originalRanks))

	for _, rank := range originalRanks {
		result = append(result, fixture.byRank[rank])
	}

	return result
}

// reorder runs Reorder on the "photo" category, naming the Attachments by the ranks they started with
func (fixture reorderFixture) reorder(originalRanks ...int) error {
	return fixture.service.Reorder(fixture.session, "Stream", fixture.objectID, "photo", fixture.ids(originalRanks...), "sorted")
}

// load reads one Attachment back from the database
func (fixture reorderFixture) load(t *testing.T, attachmentID primitive.ObjectID) model.Attachment {
	t.Helper()

	attachment := model.NewEmptyAttachment()
	require.NoError(t, fixture.session.Collection("Attachment").Load(exp.Equal("_id", attachmentID), &attachment))

	return attachment
}

// rankOf reads the stored rank of the Attachment that started at the given rank
func (fixture reorderFixture) rankOf(t *testing.T, originalRank int) int {
	t.Helper()
	return fixture.load(t, fixture.byRank[originalRank]).Rank
}

// revisionOf reads how many times the Attachment that started at the given rank has been saved since it was stored
func (fixture reorderFixture) revisionOf(t *testing.T, originalRank int) int64 {
	t.Helper()
	return fixture.load(t, fixture.byRank[originalRank]).Revision
}

// TestAttachment_Reorder_Reverses gives the named Attachments their own ranks in the new order
func TestAttachment_Reorder_Reverses(t *testing.T) {

	fixture := newReorderFixture(t, 1, 2, 3)
	require.NoError(t, fixture.reorder(3, 2, 1))

	require.Equal(t, 1, fixture.rankOf(t, 3))
	require.Equal(t, 2, fixture.rankOf(t, 2))
	require.Equal(t, 3, fixture.rankOf(t, 1))
}

// TestAttachment_Reorder_PartialOrderKeepsTheRest trades ranks only among the Attachments named
func TestAttachment_Reorder_PartialOrderKeepsTheRest(t *testing.T) {

	fixture := newReorderFixture(t, 1, 2, 3, 4)
	require.NoError(t, fixture.reorder(3, 1))

	require.Equal(t, 1, fixture.rankOf(t, 3))
	require.Equal(t, 2, fixture.rankOf(t, 2))
	require.Equal(t, 3, fixture.rankOf(t, 1))
	require.Equal(t, 4, fixture.rankOf(t, 4))
}

// TestAttachment_Reorder_GapsAreKept hands out the ranks the Attachments held, not consecutive ones
func TestAttachment_Reorder_GapsAreKept(t *testing.T) {

	fixture := newReorderFixture(t, 10, 20, 30)
	require.NoError(t, fixture.reorder(30, 10, 20))

	require.Equal(t, 10, fixture.rankOf(t, 30))
	require.Equal(t, 20, fixture.rankOf(t, 10))
	require.Equal(t, 30, fixture.rankOf(t, 20))
}

// TestAttachment_Reorder_RenumbersTies numbers Attachments that all hold 0 before trading, so the
// named ones take 1 to n in the order named
func TestAttachment_Reorder_RenumbersTies(t *testing.T) {

	fixture := newReorderFixture(t)
	first := fixture.store(t, "Stream", fixture.objectID, "photo", 0)
	second := fixture.store(t, "Stream", fixture.objectID, "photo", 0)
	third := fixture.store(t, "Stream", fixture.objectID, "photo", 0)

	require.NoError(t, fixture.service.Reorder(fixture.session, "Stream", fixture.objectID, "photo", []primitive.ObjectID{third, first, second}, "sorted"))

	require.Equal(t, 1, fixture.load(t, third).Rank)
	require.Equal(t, 2, fixture.load(t, first).Rank)
	require.Equal(t, 3, fixture.load(t, second).Rank)
}

// TestAttachment_Reorder_SkipsUnknownIDs ignores an ID that is not in the category, on this object
func TestAttachment_Reorder_SkipsUnknownIDs(t *testing.T) {

	fixture := newReorderFixture(t, 1, 2)
	cover := fixture.store(t, "Stream", fixture.objectID, "cover", 1)
	elsewhere := fixture.store(t, "Stream", primitive.NewObjectID(), "photo", 1)
	user := fixture.store(t, "User", fixture.objectID, "photo", 1)

	order := []primitive.ObjectID{primitive.NewObjectID(), cover, elsewhere, user, fixture.byRank[2], fixture.byRank[1]}
	require.NoError(t, fixture.service.Reorder(fixture.session, "Stream", fixture.objectID, "photo", order, "sorted"))

	require.Equal(t, 1, fixture.rankOf(t, 2))
	require.Equal(t, 2, fixture.rankOf(t, 1))
	require.Equal(t, 1, fixture.load(t, cover).Rank)
	require.Equal(t, 1, fixture.load(t, elsewhere).Rank)
	require.Equal(t, 1, fixture.load(t, user).Rank)
}

// TestAttachment_Reorder_SkipsDeleted leaves a deleted Attachment out of the trade
func TestAttachment_Reorder_SkipsDeleted(t *testing.T) {

	fixture := newReorderFixture(t, 1, 2, 3)

	deleted := fixture.load(t, fixture.byRank[2])
	require.NoError(t, fixture.session.Collection("Attachment").Delete(&deleted, "gone"))

	require.NoError(t, fixture.reorder(3, 2, 1))

	require.Equal(t, 1, fixture.rankOf(t, 3))
	require.Equal(t, 2, fixture.rankOf(t, 2))
	require.Equal(t, 3, fixture.rankOf(t, 1))
}

// TestAttachment_Reorder_EmptyOrderChangesNothing saves nothing when no Attachment is named
func TestAttachment_Reorder_EmptyOrderChangesNothing(t *testing.T) {

	fixture := newReorderFixture(t, 1, 2, 3)
	stored := fixture.revisionOf(t, 1)

	require.NoError(t, fixture.service.Reorder(fixture.session, "Stream", fixture.objectID, "photo", nil, "sorted"))

	for _, rank := range []int{1, 2, 3} {
		require.Equal(t, rank, fixture.rankOf(t, rank))
		require.Equal(t, stored, fixture.revisionOf(t, rank))
	}
}

// TestAttachment_Reorder_SavesOnlyWhatMoved leaves the journal of an unmoved Attachment alone
func TestAttachment_Reorder_SavesOnlyWhatMoved(t *testing.T) {

	fixture := newReorderFixture(t, 1, 2, 3)
	stored := fixture.revisionOf(t, 1)

	require.NoError(t, fixture.reorder(1, 2, 3))
	require.Equal(t, stored, fixture.revisionOf(t, 1))
	require.Equal(t, stored, fixture.revisionOf(t, 2))
	require.Equal(t, stored, fixture.revisionOf(t, 3))

	require.NoError(t, fixture.reorder(2, 1, 3))
	require.Equal(t, stored+1, fixture.revisionOf(t, 1))
	require.Equal(t, stored+1, fixture.revisionOf(t, 2))
	require.Equal(t, stored, fixture.revisionOf(t, 3))
}

// TestAttachment_Reorder_SaveFails returns the first save's error
func TestAttachment_Reorder_SaveFails(t *testing.T) {

	fixture := newReorderFixture(t, 1)
	invalid := fixture.storeWithStatus(t, "Stream", fixture.objectID, "photo", 2, "bogus")

	require.Error(t, fixture.service.Reorder(fixture.session, "Stream", fixture.objectID, "photo", []primitive.ObjectID{invalid, fixture.byRank[1]}, "sorted"))
}

// TestAttachment_Reorder_EveryCategory trades across categories when none is named
func TestAttachment_Reorder_EveryCategory(t *testing.T) {

	fixture := newReorderFixture(t, 1)
	cover := fixture.store(t, "Stream", fixture.objectID, "cover", 2)

	require.NoError(t, fixture.service.Reorder(fixture.session, "Stream", fixture.objectID, "", []primitive.ObjectID{cover, fixture.byRank[1]}, "sorted"))

	require.Equal(t, 1, fixture.load(t, cover).Rank)
	require.Equal(t, 2, fixture.rankOf(t, 1))
}

// TestReorderRanks pins the ranks reorderRanks hands out, by the position of each Attachment
func TestReorderRanks(t *testing.T) {

	// check builds Attachments with the given ranks, names them by index in `order` (an index
	// past the end names an Attachment that does not exist), and compares the ranks returned
	check := func(name string, ranks []int, order []int, expected []int) {
		t.Helper()

		attachments := make([]model.Attachment, len(ranks))

		for index, rank := range ranks {
			attachments[index] = model.NewAttachment("Stream", primitive.NilObjectID)
			attachments[index].Rank = rank
		}

		named := make([]primitive.ObjectID, 0, len(order))

		for _, index := range order {
			if index < len(attachments) {
				named = append(named, attachments[index].AttachmentID)
			} else {
				named = append(named, primitive.NewObjectID())
			}
		}

		result := reorderRanks(attachments, named)
		require.Len(t, result, len(attachments), name)

		actual := make([]int, len(attachments))

		for index, attachment := range attachments {
			actual[index] = result[attachment.AttachmentID]
		}

		require.Equal(t, expected, actual, name)
	}

	check("nothing", []int{}, []int{}, []int{})
	check("no order keeps ranks", []int{1, 2, 3}, []int{}, []int{1, 2, 3})
	check("the same order keeps ranks", []int{1, 2, 3}, []int{0, 1, 2}, []int{1, 2, 3})
	check("reverse", []int{1, 2, 3}, []int{2, 1, 0}, []int{3, 2, 1})
	check("a partial order trades only among itself", []int{1, 2, 3, 4}, []int{2, 0}, []int{3, 2, 1, 4})
	check("gaps are kept", []int{10, 20, 30}, []int{2, 0, 1}, []int{20, 30, 10})
	check("an unknown id is skipped", []int{1, 2}, []int{5, 1, 0}, []int{2, 1})
	check("a repeated id counts once", []int{1, 2, 3}, []int{2, 2, 0}, []int{3, 2, 1})
	check("one attachment keeps its rank", []int{7}, []int{0}, []int{7})

	// Ranks that cannot be traded are renumbered in their current order first
	check("unset ranks are renumbered", []int{0, 0, 0}, []int{}, []int{1, 2, 3})
	check("unset ranks are renumbered, then traded", []int{0, 0, 0}, []int{2, 1, 0}, []int{3, 2, 1})
	check("one unset rank is renumbered", []int{0}, []int{0}, []int{1})
	check("a shared rank renumbers everything", []int{1, 1, 2}, []int{}, []int{1, 2, 3})
	check("a negative rank renumbers everything", []int{-1, 5}, []int{}, []int{1, 2})
	check("one unset rank among many renumbers everything", []int{3, 0, 9}, []int{2, 0, 1}, []int{2, 3, 1})
}

// FuzzReorderRanks checks that the ranks handed out are the ranks held, and that the named
// Attachments come out in the order named
func FuzzReorderRanks(f *testing.F) {

	f.Add([]byte{1, 2, 3}, []byte{2, 1, 0})
	f.Add([]byte{0, 0, 0}, []byte{1})
	f.Add([]byte{5, 5, 1}, []byte{9, 0, 0, 2})
	f.Add([]byte{}, []byte{})

	f.Fuzz(func(t *testing.T, ranks []byte, order []byte) {

		attachments := make([]model.Attachment, len(ranks))

		for index, rank := range ranks {
			attachments[index] = model.NewAttachment("Stream", primitive.NilObjectID)
			attachments[index].Rank = int(int8(rank)) // #nosec G115 -- negative ranks are a case under test
		}

		named := make([]primitive.ObjectID, 0, len(order))

		for _, index := range order {
			if int(index) < len(attachments) {
				named = append(named, attachments[index].AttachmentID)
			}
		}

		result := reorderRanks(attachments, named)
		require.Len(t, result, len(attachments))

		// The ranks handed out are the ranks held, or 1 to n when those could not be traded
		after := make([]int, 0, len(attachments))

		for _, attachment := range attachments {
			after = append(after, result[attachment.AttachmentID])
		}

		require.ElementsMatch(t, tradableRanks(attachments), after)

		// The named Attachments come out in the order named, each with a higher rank than the last
		last := 0

		for _, attachmentID := range firstMentions(named) {
			require.Greater(t, result[attachmentID], last)
			last = result[attachmentID]
		}
	})
}

// tradableRanks returns the ranks reorderRanks may hand out: the ranks held when they are
// distinct and set, and 1 to n otherwise
func tradableRanks(attachments []model.Attachment) []int {

	result := make([]int, len(attachments))

	for index, attachment := range attachments {
		if ranksAreDistinct(attachments) {
			result[index] = attachment.Rank
		} else {
			result[index] = index + 1
		}
	}

	return result
}

// firstMentions returns the IDs in the order first named, once each
func firstMentions(named []primitive.ObjectID) []primitive.ObjectID {

	result := make([]primitive.ObjectID, 0, len(named))
	seen := map[primitive.ObjectID]bool{}

	for _, attachmentID := range named {

		if seen[attachmentID] {
			continue
		}

		seen[attachmentID] = true
		result = append(result, attachmentID)
	}

	return result
}
