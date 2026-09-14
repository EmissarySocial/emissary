package service

import (
	"context"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/uri"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// One invariant, checked for every actor type Emissary publishes: the WebFinger subject is exactly
// "acct:" + the actor document's preferredUsername + "@" + hostname. A remote server reconstructs
// that handle from the actor document and asks WebFinger for it, so the two must never disagree
// (BUG-98). Stream is covered by TestStreamWebFinger_MatchesActorDocument. The Application and
// SearchDomain documents need a public key from the database, so those two cases assert against the
// same expression their JSON-LD builders use rather than against the rendered document.

/******************************************
 * In-Memory Fakes
 ******************************************/

// searchQueryCollection is an in-memory data.Collection holding one model.SearchQuery, matched on
// the fields LoadByID and notDeleted build: _id and deleteDate.
type searchQueryCollection struct {
	record model.SearchQuery
}

// Context implements the interface, returning a background context
func (c *searchQueryCollection) Context() context.Context { return context.Background() }

// Count implements the data.Collection interface. Unused by these tests.
func (c *searchQueryCollection) Count(exp.Expression, ...option.Option) (int64, error) {
	return 0, derp.Internal("test", "unused")
}

// Query implements the data.Collection interface. Unused by these tests.
func (c *searchQueryCollection) Query(any, exp.Expression, ...option.Option) error {
	return derp.Internal("test", "unused")
}

// Iterator implements the data.Collection interface. Unused by these tests.
func (c *searchQueryCollection) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.Internal("test", "unused")
}

// Load copies the stored SearchQuery into the target when the criteria match.
func (c *searchQueryCollection) Load(criteria exp.Expression, target data.Object, _ ...option.Option) error {

	searchQuery, ok := target.(*model.SearchQuery)

	if !ok {
		return derp.Internal("test", "unexpected target type")
	}

	matched := criteria.Match(func(predicate exp.Predicate) bool {

		if predicate.Operator != exp.OperatorEqual {
			return false
		}

		switch predicate.Field {

		case "_id":
			value, ok := predicate.Value.(primitive.ObjectID)
			return ok && c.record.SearchQueryID == value

		case "deleteDate":
			value, ok := predicate.Value.(int)
			return ok && c.record.DeleteDate == int64(value)

		default:
			return false
		}
	})

	if !matched {
		return derp.NotFound("test", "not found")
	}

	*searchQuery = c.record
	return nil
}

// Save implements the data.Collection interface. Unused by these tests.
func (c *searchQueryCollection) Save(data.Object, string) error {
	return derp.Internal("test", "unused")
}

// Delete implements the data.Collection interface. Unused by these tests.
func (c *searchQueryCollection) Delete(data.Object, string) error {
	return derp.Internal("test", "unused")
}

// HardDelete implements the data.Collection interface. Unused by these tests.
func (c *searchQueryCollection) HardDelete(exp.Expression) error {
	return derp.Internal("test", "unused")
}

/******************************************
 * Tests
 ******************************************/

// TestWebFingerInvariant_User checks the subject against the rendered actor document
func TestWebFingerInvariant_User(t *testing.T) {

	user := model.NewUser()
	user.UserID = primitive.NewObjectID()
	user.Username = "qapub"
	user.ProfileURL = "https://example.com/@qapub"
	user.IsPublic = true

	service, session := newWebFingerTestService(user)

	resource, err := service.WebFinger(session, "qapub")
	require.NoError(t, err)

	preferredUsername := user.GetJSONLD()[vocab.PropertyPreferredUsername]
	require.Equal(t, "acct:"+preferredUsername.(string)+"@"+uri.Hostname(service.host), resource.Subject)
}

// TestWebFingerInvariant_Application checks the subject against the literal that Domain.GetJSONLD
// writes into preferredUsername ("application"), because the rendered document needs a database.
func TestWebFingerInvariant_Application(t *testing.T) {

	service := &Domain{host: "https://example.com", hostname: "example.com"}

	require.Equal(t, "acct:application@"+service.Hostname(), service.WebFinger().Subject)
}

// TestWebFingerInvariant_SearchDomain checks the subject against ActivityPubUsername, which is the
// expression SearchDomain.GetJSONLD writes into preferredUsername.
func TestWebFingerInvariant_SearchDomain(t *testing.T) {

	service := &SearchDomain{host: "https://example.com"}

	require.Equal(t, "acct:"+service.ActivityPubUsername()+"@"+service.Hostname(), service.WebFinger().Subject)
}

// TestWebFingerInvariant_SearchQuery checks the subject against ActivityPubUsername, which is the
// expression SearchQuery.GetJSONLD writes into preferredUsername.
func TestWebFingerInvariant_SearchQuery(t *testing.T) {

	searchQuery := model.NewSearchQuery()
	searchQuery.URL = "https://example.com/search?q=test"

	service := &SearchQuery{host: "https://example.com"}
	session := webfingerSession{
		users:         &userCollection{},
		streams:       &streamCollection{},
		searchQueries: &searchQueryCollection{record: searchQuery},
	}

	resource, err := service.WebFinger(session, searchQuery.SearchQueryID.Hex())
	require.NoError(t, err)

	expected := "acct:" + service.ActivityPubUsername(searchQuery.SearchQueryID) + "@" + service.Hostname()
	require.Equal(t, expected, resource.Subject)
}
