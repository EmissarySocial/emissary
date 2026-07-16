package sync

import (
	"testing"

	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// newResponseRecord builds one reaction in a group, stamped with a createDate so that the
// newest-wins rule has something to sort on.
func newResponseRecord(responseType string, createDate int64) responseRecord {
	return responseRecord{
		ResponseID: primitive.NewObjectID(),
		Type:       responseType,
		CreateDate: createDate,
	}
}

/******************************************
 * displacedResponseIDs
 ******************************************/

// BUG-003249: repeated Likes each inserted a row. Cleanup keeps only the newest.
func TestDisplacedResponseIDs_DuplicateLikes(t *testing.T) {

	oldest := newResponseRecord(vocab.ActivityTypeLike, 100)
	middle := newResponseRecord(vocab.ActivityTypeLike, 200)
	newest := newResponseRecord(vocab.ActivityTypeLike, 300)

	group := responseGroup{Responses: []responseRecord{oldest, middle, newest}}
	result := group.displacedResponseIDs()

	require.ElementsMatch(t, []primitive.ObjectID{oldest.ResponseID, middle.ResponseID}, result)
}

// A Like and a Dislike cannot coexist: the newer reaction wins.
func TestDisplacedResponseIDs_LikeAndDislikeContradiction(t *testing.T) {

	like := newResponseRecord(vocab.ActivityTypeLike, 100)
	dislike := newResponseRecord(vocab.ActivityTypeDislike, 200)

	group := responseGroup{Responses: []responseRecord{like, dislike}}
	result := group.displacedResponseIDs()

	require.Equal(t, []primitive.ObjectID{like.ResponseID}, result)
}

// The rule is symmetric -- an older Dislike loses to a newer Like just the same.
func TestDisplacedResponseIDs_DislikeAndLikeContradiction(t *testing.T) {

	dislike := newResponseRecord(vocab.ActivityTypeDislike, 100)
	like := newResponseRecord(vocab.ActivityTypeLike, 200)

	group := responseGroup{Responses: []responseRecord{dislike, like}}
	result := group.displacedResponseIDs()

	require.Equal(t, []primitive.ObjectID{dislike.ResponseID}, result)
}

// An Announce survives alongside a Like -- they do not displace each other.
func TestDisplacedResponseIDs_AnnounceCoexistsWithLike(t *testing.T) {

	group := responseGroup{Responses: []responseRecord{
		newResponseRecord(vocab.ActivityTypeLike, 100),
		newResponseRecord(vocab.ActivityTypeAnnounce, 200),
	}}

	require.Empty(t, group.displacedResponseIDs())
}

// Duplicate Announces still collapse to the newest, even though Announce conflicts with nothing else.
func TestDisplacedResponseIDs_DuplicateAnnounces(t *testing.T) {

	oldest := newResponseRecord(vocab.ActivityTypeAnnounce, 100)
	newest := newResponseRecord(vocab.ActivityTypeAnnounce, 200)

	group := responseGroup{Responses: []responseRecord{oldest, newest}}

	require.Equal(t, []primitive.ObjectID{oldest.ResponseID}, group.displacedResponseIDs())
}

// The reported reproduction: 3 Likes + 1 Dislike on one post. Exactly one reaction survives.
func TestDisplacedResponseIDs_ReportedReproduction(t *testing.T) {

	group := responseGroup{Responses: []responseRecord{
		newResponseRecord(vocab.ActivityTypeLike, 100),
		newResponseRecord(vocab.ActivityTypeLike, 200),
		newResponseRecord(vocab.ActivityTypeLike, 300),
		newResponseRecord(vocab.ActivityTypeDislike, 400),
	}}

	require.Len(t, group.displacedResponseIDs(), 3)
}

// Duplicates and an independent Announce at once: the Likes collapse, the Announce survives.
func TestDisplacedResponseIDs_MixedBuckets(t *testing.T) {

	survivingLike := newResponseRecord(vocab.ActivityTypeLike, 300)
	announce := newResponseRecord(vocab.ActivityTypeAnnounce, 400)

	group := responseGroup{Responses: []responseRecord{
		newResponseRecord(vocab.ActivityTypeLike, 100),
		newResponseRecord(vocab.ActivityTypeDislike, 200),
		survivingLike,
		announce,
	}}

	result := group.displacedResponseIDs()

	require.Len(t, result, 2)
	require.NotContains(t, result, survivingLike.ResponseID)
	require.NotContains(t, result, announce.ResponseID)
}

// A lone reaction is never displaced. The aggregation filters these out, but the rule must hold.
func TestDisplacedResponseIDs_SingleResponse(t *testing.T) {

	group := responseGroup{Responses: []responseRecord{
		newResponseRecord(vocab.ActivityTypeLike, 100),
	}}

	require.Empty(t, group.displacedResponseIDs())
}

// An empty group must not panic (displacedResponseIDs indexes into each bucket it builds).
func TestDisplacedResponseIDs_Empty(t *testing.T) {
	require.Empty(t, responseGroup{}.displacedResponseIDs())
	require.Empty(t, responseGroup{Responses: []responseRecord{}}.displacedResponseIDs())
}

// Contradictions created within the same millisecond still collapse to exactly one survivor.
func TestDisplacedResponseIDs_TiedCreateDates(t *testing.T) {

	group := responseGroup{Responses: []responseRecord{
		newResponseRecord(vocab.ActivityTypeLike, 100),
		newResponseRecord(vocab.ActivityTypeDislike, 100),
	}}

	require.Len(t, group.displacedResponseIDs(), 1)
}

// Every surviving reaction leaves exactly one row per conflict bucket, which is what lets the
// unique (userId, object, type) index build.
func TestDisplacedResponseIDs_LeavesOneSurvivorPerBucket(t *testing.T) {

	group := responseGroup{Responses: []responseRecord{
		newResponseRecord(vocab.ActivityTypeLike, 100),
		newResponseRecord(vocab.ActivityTypeLike, 200),
		newResponseRecord(vocab.ActivityTypeDislike, 300),
		newResponseRecord(vocab.ActivityTypeAnnounce, 400),
		newResponseRecord(vocab.ActivityTypeAnnounce, 500),
	}}

	// 5 reactions in 2 buckets ({Like,Dislike} and {Announce}) leave 2 survivors
	require.Len(t, group.displacedResponseIDs(), 3)
}

/******************************************
 * newestResponseID
 ******************************************/

// The newest record wins regardless of its position in the bucket.
func TestNewestResponseID(t *testing.T) {

	first := newResponseRecord(vocab.ActivityTypeLike, 300)
	middle := newResponseRecord(vocab.ActivityTypeLike, 100)
	last := newResponseRecord(vocab.ActivityTypeLike, 200)

	require.Equal(t, first.ResponseID, newestResponseID([]responseRecord{first, middle, last}))
	require.Equal(t, first.ResponseID, newestResponseID([]responseRecord{middle, last, first}))
	require.Equal(t, first.ResponseID, newestResponseID([]responseRecord{middle, first, last}))
}

// A single-record bucket returns that record.
func TestNewestResponseID_Single(t *testing.T) {
	only := newResponseRecord(vocab.ActivityTypeLike, 100)
	require.Equal(t, only.ResponseID, newestResponseID([]responseRecord{only}))
}
