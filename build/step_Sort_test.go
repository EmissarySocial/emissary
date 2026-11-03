package build

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// stubSortService is a ModelService that holds Streams in memory and records each save.
// Any other ModelService call panics through the nil embedded interface, because StepSort makes none.
type stubSortService struct {
	service.ModelService
	streams  map[primitive.ObjectID]*model.Stream
	saved    []primitive.ObjectID
	failSave bool
}

// ObjectLoad returns the Stream whose ID matches the criteria. Implements the ModelService interface.
func (stub *stubSortService) ObjectLoad(_ data.Session, criteria exp.Expression) (data.Object, error) {

	predicate, isPredicate := criteria.(exp.Predicate)

	if !isPredicate {
		return nil, derp.Internal("stubSortService.ObjectLoad", "Criteria must be a single predicate")
	}

	streamID, _ := predicate.Value.(primitive.ObjectID)
	stream, exists := stub.streams[streamID]

	if !exists {
		return nil, derp.NotFound("stubSortService.ObjectLoad", "Stream not found", streamID)
	}

	return stream, nil
}

// ObjectSave records which Stream was saved. Implements the ModelService interface.
func (stub *stubSortService) ObjectSave(_ data.Session, object data.Object, _ string) error {

	if stub.failSave {
		return derp.Internal("stubSortService.ObjectSave", "Save refused")
	}

	stub.saved = append(stub.saved, object.(*model.Stream).StreamID)
	return nil
}

// Schema returns the Stream schema. Implements the ModelService interface.
func (stub *stubSortService) Schema() schema.Schema {
	return schema.New(model.StreamSchema())
}

// stubSortBuilder satisfies the Builder interface by embedding it (nil) and overriding only the
// three methods StepSort reaches for.
type stubSortBuilder struct {
	Builder
	_request *http.Request
	_service service.ModelService
}

// request returns the POST being sorted. Implements the Builder interface.
func (builder stubSortBuilder) request() *http.Request {
	return builder._request
}

// service returns the stubbed model service. Implements the Builder interface.
func (builder stubSortBuilder) service() service.ModelService {
	return builder._service
}

// stubSortSession is a data.Session that the stubbed service never calls
type stubSortSession struct {
	data.Session
}

// session returns an empty session, which the stubbed service ignores. Implements the Builder interface.
func (builder stubSortBuilder) session() data.Session {
	return stubSortSession{}
}

// sortFixture holds Streams with the given ranks, keyed by their rank for readable assertions.
type sortFixture struct {
	service *stubSortService
	byRank  map[int]*model.Stream
}

// newSortFixture creates one Stream for each rank given
func newSortFixture(ranks ...int) sortFixture {

	result := sortFixture{
		service: &stubSortService{streams: map[primitive.ObjectID]*model.Stream{}},
		byRank:  map[int]*model.Stream{},
	}

	for _, rank := range ranks {
		stream := model.NewStream()
		stream.Rank = rank
		result.service.streams[stream.StreamID] = &stream
		result.byRank[rank] = &stream
	}

	return result
}

// post runs a sort step against the Streams with the given original ranks, in the order given
func (fixture sortFixture) post(t *testing.T, direction string, ranks ...int) PipelineResult {
	t.Helper()

	values := url.Values{}
	for _, rank := range ranks {
		values.Add("keys", fixture.id(t, rank))
	}

	return fixture.postValues(direction, values)
}

// postValues runs a sort step against a raw form body
func (fixture sortFixture) postValues(direction string, values url.Values) PipelineResult {

	request := httptest.NewRequest(http.MethodPost, "/000000000000000000000000/sort", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	step := StepSort{Keys: "_id", Values: "rank", Direction: direction}
	builder := stubSortBuilder{_request: request, _service: fixture.service}

	return applyBehavior(step.Post(builder, nil))
}

// stream returns the Stream that started at the given rank, failing the test if there is none
func (fixture sortFixture) stream(t *testing.T, originalRank int) *model.Stream {
	t.Helper()

	if stream, exists := fixture.byRank[originalRank]; exists && stream != nil {
		return stream
	}

	t.Fatalf("no Stream started at rank %d", originalRank)
	return &model.Stream{}
}

// id returns the hex ID of the Stream that started at the given rank
func (fixture sortFixture) id(t *testing.T, originalRank int) string {
	t.Helper()
	return fixture.stream(t, originalRank).StreamID.Hex()
}

// rankOf returns the current rank of the Stream that started at the given rank
func (fixture sortFixture) rankOf(t *testing.T, originalRank int) int {
	t.Helper()
	return fixture.stream(t, originalRank).Rank
}

// savedRanks returns the original ranks of the Streams that were saved, in save order
func (fixture sortFixture) savedRanks() []int {

	result := make([]int, 0, len(fixture.service.saved))

	for _, streamID := range fixture.service.saved {
		for rank, stream := range fixture.byRank {
			if stream.StreamID == streamID {
				result = append(result, rank)
			}
		}
	}

	return result
}

// TestStepSort_Ascending numbers the posted records from 1, in the order posted
func TestStepSort_Ascending(t *testing.T) {

	fixture := newSortFixture(1, 2, 3)
	result := fixture.post(t, "asc", 3, 1, 2)

	require.Nil(t, result.Error)
	require.False(t, result.Halt)
	require.Equal(t, "none", result.Headers.GetString("HX-Reswap"))
	require.Equal(t, 1, fixture.rankOf(t, 3))
	require.Equal(t, 2, fixture.rankOf(t, 1))
	require.Equal(t, 3, fixture.rankOf(t, 2))
	require.Equal(t, []int{3, 1, 2}, fixture.savedRanks())
}

// TestStepSort_SkipsUnchanged saves only the records whose rank moved
func TestStepSort_SkipsUnchanged(t *testing.T) {

	fixture := newSortFixture(1, 2, 3, 4)
	result := fixture.post(t, "asc", 1, 3, 2, 4)

	require.Nil(t, result.Error)
	require.Equal(t, []int{3, 2}, fixture.savedRanks())
	require.Equal(t, 1, fixture.rankOf(t, 1))
	require.Equal(t, 2, fixture.rankOf(t, 3))
	require.Equal(t, 3, fixture.rankOf(t, 2))
	require.Equal(t, 4, fixture.rankOf(t, 4))
}

// TestStepSort_SameOrder saves nothing when nothing moved
func TestStepSort_SameOrder(t *testing.T) {

	fixture := newSortFixture(1, 2, 3)
	result := fixture.post(t, "asc", 1, 2, 3)

	require.Nil(t, result.Error)
	require.Empty(t, fixture.service.saved)
}

// TestStepSort_Descending numbers the posted records down from the highest posted rank
func TestStepSort_Descending(t *testing.T) {

	fixture := newSortFixture(10, 9, 8, 7)

	// Drag the bottom photo up to second place
	result := fixture.post(t, "desc", 10, 7, 9, 8)

	require.Nil(t, result.Error)
	require.Equal(t, "none", result.Headers.GetString("HX-Reswap"))
	require.Equal(t, 10, fixture.rankOf(t, 10))
	require.Equal(t, 9, fixture.rankOf(t, 7))
	require.Equal(t, 8, fixture.rankOf(t, 9))
	require.Equal(t, 7, fixture.rankOf(t, 8))
	require.Equal(t, []int{7, 9, 8}, fixture.savedRanks())
}

// TestStepSort_DescendingPartialList keeps the posted page above the records it did not post
func TestStepSort_DescendingPartialList(t *testing.T) {

	// Ranks 1 through 5 exist but are not loaded, as on an infinite scroll
	fixture := newSortFixture(100, 75, 99, 1, 2, 3, 4, 5)
	result := fixture.post(t, "desc", 75, 100, 99)

	require.Nil(t, result.Error)
	require.Equal(t, 100, fixture.rankOf(t, 75))
	require.Equal(t, 99, fixture.rankOf(t, 100))
	require.Equal(t, 98, fixture.rankOf(t, 99))

	for _, unposted := range []int{1, 2, 3, 4, 5} {
		require.Equal(t, unposted, fixture.rankOf(t, unposted), "unposted rank %d", unposted)
	}
}

// TestStepSort_DescendingUnranked numbers records that have no rank yet down from the list's length
func TestStepSort_DescendingUnranked(t *testing.T) {

	fixture := newSortFixture(0)
	extra := model.NewStream()
	fixture.service.streams[extra.StreamID] = &extra

	values := url.Values{"keys": {fixture.id(t, 0), extra.StreamID.Hex()}}
	result := fixture.postValues("desc", values)

	require.Nil(t, result.Error)
	require.Equal(t, 2, fixture.rankOf(t, 0))
	require.Equal(t, 1, extra.Rank)
}

// TestStepSort_NoKeys saves nothing and still suppresses the swap
func TestStepSort_NoKeys(t *testing.T) {

	fixture := newSortFixture(1, 2)
	result := fixture.postValues("desc", url.Values{})

	require.Nil(t, result.Error)
	require.False(t, result.Halt)
	require.Equal(t, "none", result.Headers.GetString("HX-Reswap"))
	require.Empty(t, fixture.service.saved)
}

// TestStepSort_InvalidKey halts before saving anything
func TestStepSort_InvalidKey(t *testing.T) {

	fixture := newSortFixture(2, 1)
	values := url.Values{"keys": {fixture.id(t, 1), "not-an-object-id"}}
	result := fixture.postValues("asc", values)

	require.NotNil(t, result.Error)
	require.True(t, result.Halt)
	require.Empty(t, fixture.service.saved)
	require.Equal(t, 1, fixture.rankOf(t, 1))
}

// TestStepSort_MissingRecord halts before saving anything, even records that loaded first
func TestStepSort_MissingRecord(t *testing.T) {

	fixture := newSortFixture(2, 1)
	values := url.Values{"keys": {fixture.id(t, 1), primitive.NewObjectID().Hex()}}
	result := fixture.postValues("asc", values)

	require.NotNil(t, result.Error)
	require.True(t, result.Halt)
	require.Empty(t, fixture.service.saved)
}

// TestStepSort_SaveFails halts with the save's error
func TestStepSort_SaveFails(t *testing.T) {

	fixture := newSortFixture(1, 2)
	fixture.service.failSave = true
	result := fixture.post(t, "asc", 2, 1)

	require.NotNil(t, result.Error)
	require.True(t, result.Halt)
}

// TestStepSort_UnsupportedBody halts with a Bad Request
func TestStepSort_UnsupportedBody(t *testing.T) {

	fixture := newSortFixture(1)
	request := httptest.NewRequest(http.MethodPost, "/000000000000000000000000/sort", strings.NewReader(`{"keys":[]}`))
	request.Header.Set("Content-Type", "application/json")

	builder := stubSortBuilder{_request: request, _service: fixture.service}
	result := applyBehavior(StepSort{Keys: "_id", Values: "rank"}.Post(builder, nil))

	require.NotNil(t, result.Error)
	require.True(t, result.Halt)
	require.Equal(t, http.StatusBadRequest, derp.ErrorCode(result.Error))
}

// TestStepSort_Get does nothing
func TestStepSort_Get(t *testing.T) {
	require.Nil(t, StepSort{}.Get(nil, nil))
}

// TestSortRanks pins the ranks assigned in each direction
func TestSortRanks(t *testing.T) {

	ranks := func(direction string, oldRanks []int, expected []int) {
		t.Helper()
		require.Equal(t, expected, sortRanks(oldRanks, direction), "%s %v", direction, oldRanks)
	}

	// Ascending ignores the current ranks
	ranks("asc", nil, []int{})
	ranks("asc", []int{}, []int{})
	ranks("asc", []int{9}, []int{1})
	ranks("asc", []int{5, 5, 5}, []int{1, 2, 3})
	ranks("asc", []int{30, 10, 20}, []int{1, 2, 3})

	// Anything but "desc" is ascending; NewSort lowercases the value first
	ranks("", []int{3, 2, 1}, []int{1, 2, 3})
	ranks("DESC", []int{3, 2, 1}, []int{1, 2, 3})

	// Descending counts down from the highest current rank
	ranks("desc", nil, []int{})
	ranks("desc", []int{7}, []int{7})
	ranks("desc", []int{3, 2, 1}, []int{3, 2, 1})
	ranks("desc", []int{1, 3, 2}, []int{3, 2, 1})
	ranks("desc", []int{75, 100, 99}, []int{100, 99, 98})
	ranks("desc", []int{1, 50}, []int{50, 49})
	ranks("desc", []int{5, 5, 5}, []int{5, 4, 3})

	// ...but never below 1, so unranked or negative records count down from the list's length
	ranks("desc", []int{0}, []int{1})
	ranks("desc", []int{0, 0, 0}, []int{3, 2, 1})
	ranks("desc", []int{-5, 2}, []int{2, 1})
	ranks("desc", []int{2, 0, 0, 0}, []int{4, 3, 2, 1})
}

// FuzzSortRanks checks that every rank is at least 1 and strictly ordered, in either direction
func FuzzSortRanks(f *testing.F) {

	f.Add([]byte{3, 2, 1}, true)
	f.Add([]byte{0, 0, 0}, true)
	f.Add([]byte{255, 0, 128}, false)
	f.Add([]byte{}, true)

	f.Fuzz(func(t *testing.T, input []byte, descending bool) {

		direction := "asc"
		if descending {
			direction = "desc"
		}

		oldRanks := make([]int, len(input))
		for index, value := range input {
			oldRanks[index] = int(int8(value)) // #nosec G115 -- negative ranks are a case under test
		}

		result := sortRanks(oldRanks, direction)
		require.Len(t, result, len(oldRanks))

		for index, rank := range result {
			require.GreaterOrEqual(t, rank, 1)

			if index == 0 {
				continue
			}

			if descending {
				require.Equal(t, result[index-1]-1, rank)
			} else {
				require.Equal(t, result[index-1]+1, rank)
			}
		}
	})
}

// stubSortFactory satisfies the Factory interface by embedding it (nil) and overriding only Model()
type stubSortFactory struct {
	Factory
	models map[string]service.ModelService
}

// Model returns the named model service, or an error if there is none. Implements the Factory interface.
func (factory stubSortFactory) Model(name string) (service.ModelService, error) {

	if result, exists := factory.models[name]; exists {
		return result, nil
	}

	return nil, derp.NotFound("stubSortFactory.Model", "Unknown model", name)
}

// stubSortFactoryBuilder is a stubSortBuilder whose factory names its model services
type stubSortFactoryBuilder struct {
	stubSortBuilder
	_factory stubSortFactory
}

// factory returns the stubbed factory. Implements the Builder interface.
func (builder stubSortFactoryBuilder) factory() Factory {
	return builder._factory
}

// TestStepSort_NamedModel sorts through the model service that the step names
func TestStepSort_NamedModel(t *testing.T) {

	fixture := newSortFixture(1, 2)
	values := url.Values{"keys": {fixture.id(t, 2), fixture.id(t, 1)}}

	request := httptest.NewRequest(http.MethodPost, "/000000000000000000000000/sort", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	builder := stubSortFactoryBuilder{
		stubSortBuilder: stubSortBuilder{_request: request},
		_factory:        stubSortFactory{models: map[string]service.ModelService{"Stream": fixture.service}},
	}

	result := applyBehavior(StepSort{Model: "Stream", Keys: "_id", Values: "rank", Direction: "asc"}.Post(builder, nil))

	require.Nil(t, result.Error)
	require.Equal(t, 1, fixture.rankOf(t, 2))
	require.Equal(t, 2, fixture.rankOf(t, 1))
}

// TestStepSort_UnknownModel halts when the named model service does not exist
func TestStepSort_UnknownModel(t *testing.T) {

	request := httptest.NewRequest(http.MethodPost, "/000000000000000000000000/sort", strings.NewReader(""))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	builder := stubSortFactoryBuilder{
		stubSortBuilder: stubSortBuilder{_request: request},
		_factory:        stubSortFactory{},
	}

	result := applyBehavior(StepSort{Model: "Nonexistent", Keys: "_id", Values: "rank"}.Post(builder, nil))

	require.NotNil(t, result.Error)
	require.True(t, result.Halt)
}

// TestStepSort_UnknownValues halts before saving when the rank property cannot be read
func TestStepSort_UnknownValues(t *testing.T) {

	fixture := newSortFixture(1)
	values := url.Values{"keys": {fixture.id(t, 1)}}

	request := httptest.NewRequest(http.MethodPost, "/000000000000000000000000/sort", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	builder := stubSortBuilder{_request: request, _service: fixture.service}
	result := applyBehavior(StepSort{Keys: "_id", Values: "notAProperty", Direction: "asc"}.Post(builder, nil))

	require.NotNil(t, result.Error)
	require.True(t, result.Halt)
	require.Empty(t, fixture.service.saved)
}

// Post's "Setting new rank" error is not covered: a property the schema can read can always be set
// to a positive integer, and sortRanks never returns less than 1.
