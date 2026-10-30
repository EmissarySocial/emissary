package service

import (
	"maps"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/camper"
	"github.com/benpate/digit"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// intentHandler names the route that completes one Activity Intent, and the transaction it binds
type intentHandler struct {
	path        string
	transaction any
}

// intentHandlers lists every Activity Intent that Emissary completes, by its WebFinger relation
var intentHandlers = map[string]intentHandler{
	camper.IntentTypeAnnounce: {path: "/@me/intent/announce", transaction: camper.AnnounceIntent{}},
	camper.IntentTypeCreate:   {path: "/@me/intent/create", transaction: camper.CreateIntent{}},
	camper.IntentTypeDislike:  {path: "/@me/intent/dislike", transaction: camper.DislikeIntent{}},
	camper.IntentTypeFollow:   {path: "/@me/intent/follow", transaction: camper.FollowIntent{}},
	camper.IntentTypeLike:     {path: "/@me/intent/like", transaction: camper.LikeIntent{}},
}

// TestUser_WebFinger_IntentLinks confirms that a public profile publishes every intent, each asking for what its handler reads
func TestUser_WebFinger_IntentLinks(t *testing.T) {

	user := model.NewUser()
	user.UserID = primitive.NewObjectID()
	user.Username = "qaintents"
	user.IsPublic = true

	service, session := newWebFingerTestService(user)

	resource, err := service.WebFinger(session, "qaintents")
	require.Nil(t, err)

	checkIntentLinks(t, resource)
}

// TestDomain_WebFinger_IntentLinks confirms that the Domain's service Actor publishes every intent, each asking for what its handler reads
func TestDomain_WebFinger_IntentLinks(t *testing.T) {

	service := &Domain{host: "https://example.com", hostname: "example.com"}

	checkIntentLinks(t, service.WebFinger())
}

// checkIntentLinks requires a valid link for every intent Emissary completes, and no link for any other
func checkIntentLinks(t *testing.T, resource digit.Resource) {

	t.Helper()

	// Every intent that Emissary completes is published, and leads to its handler
	for intentType, handler := range intentHandlers {
		t.Run(strings.TrimPrefix(intentType, digit.RelationTypeFEP3b86Prefix), func(t *testing.T) {
			requireIntentLink(t, resource, intentType, handler)
		})
	}

	// RULE: An intent with no handler is never published, or a remote server would send visitors to a dead end
	for _, link := range resource.Links {
		if strings.HasPrefix(link.RelationType, digit.RelationTypeFEP3b86Prefix) {
			_, isHandled := intentHandlers[link.RelationType]
			require.True(t, isHandled, "no handler completes the published %s intent", link.RelationType)
		}
	}
}

// requireIntentLink fails unless the resource publishes intentType as a template on the handler's route,
// whose query asks for exactly the fields that the handler binds from it
func requireIntentLink(t *testing.T, resource digit.Resource, intentType string, handler intentHandler) {

	t.Helper()

	// Find the link, which digit keeps in Template rather than Href
	link := resource.FindLink(intentType)
	require.NotEmpty(t, link.Template, "%s is not published", intentType)

	parsed, err := url.Parse(link.Template)
	require.NoError(t, err)
	require.Equal(t, "https://example.com"+handler.path, parsed.Scheme+"://"+parsed.Host+parsed.Path)

	// RULE: The template asks for every field the handler binds, and nothing else
	query, err := url.ParseQuery(parsed.RawQuery)
	require.NoError(t, err)
	require.ElementsMatch(t, queryFields(handler.transaction), slices.Collect(maps.Keys(query)))

	// Each field is a placeholder of its own name, for the remote server to fill in
	for name := range query {
		require.Equal(t, "{"+name+"}", query.Get(name), "the %q placeholder", name)
	}
}

// queryFields returns the query tag of every field in an intent transaction, which is what its GET handler binds
func queryFields(transaction any) []string {

	transactionType := reflect.TypeOf(transaction)
	result := make([]string, 0, transactionType.NumField())

	for index := range transactionType.NumField() {
		result = append(result, transactionType.Field(index).Tag.Get("query"))
	}

	return result
}
