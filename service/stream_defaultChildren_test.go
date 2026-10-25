package service

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/realtime"
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// newDefaultChildTemplates returns the Templates these tests share: an outbox that declares the
// given children, a collection that an outbox can hold, and a page that it cannot
func newDefaultChildTemplates(entries ...mapof.Any) set.Map[model.Template] {

	collection := model.NewTemplate("test-collection", nil)
	collection.TemplateRole = "test-collection"
	collection.Model = "Stream"
	collection.ContainedBy = sliceof.String{"outbox"}
	collection.States = mapof.Object[model.State]{"default": {StateID: "default"}}
	collection.AccessRoles = mapof.Object[model.Role]{"viewer": {RoleID: "viewer"}}
	collection.Actions = mapof.Object[model.Action]{"view": {
		Roles:      sliceof.String{"author", "viewer"},
		AccessList: mapof.Object[sliceof.String]{"default": {"author", "viewer"}},
	}}
	collection.Schema = schema.New(schema.Object{Properties: schema.ElementMap{
		"data": schema.Object{Properties: schema.ElementMap{
			"layout": schema.String{Enum: []string{"column-3", "notes"}},
			"zoom":   schema.Integer{},
		}},
	}})
	collection.Schema.Inherit(schema.New(collection.BaseSchema()))

	page := model.NewTemplate("test-page", nil)
	page.TemplateRole = "test-page"
	page.Model = "Stream"
	page.ContainedBy = sliceof.String{"top"}
	page.States = mapof.Object[model.State]{"default": {StateID: "default"}}
	page.Schema = schema.New(page.BaseSchema())

	outbox := model.NewTemplate("test-outbox", nil)
	outbox.TemplateRole = "user-outbox"
	outbox.Model = "Outbox"
	outbox.DefaultChildren = entries

	return set.Map[model.Template]{
		collection.TemplateID: collection,
		page.TemplateID:       page,
		outbox.TemplateID:     outbox,
	}
}

// newDefaultChildService returns a Stream service whose live Templates are newDefaultChildTemplates
func newDefaultChildService(entries ...mapof.Any) *Stream {
	return &Stream{
		host:            "https://example.test",
		templateService: &Template{templates: newDefaultChildTemplates(entries...)},
	}
}

// newDefaultChildUser returns a new User whose profile uses the test outbox
func newDefaultChildUser() model.User {
	user := model.NewUser()
	user.Username = "sarah"
	user.DisplayName = "Sarah Connor"
	user.ProfileURL = "https://example.test/@" + user.UserID.Hex()
	user.OutboxTemplate = "test-outbox"
	return user
}

// articlesEntry returns a default child shaped like qwertylicious' Articles collection
func articlesEntry() mapof.Any {
	return mapof.Any{
		"templateId":      "test-collection",
		"label":           "Articles",
		"data.layout":     "column-3",
		"data.zoom":       "11",
		"replyPolicy":     "everyone",
		"groups.viewer.0": "000000000000000000000000",
	}
}

// notesEntry returns a default child shaped like qwertylicious' Notes collection
func notesEntry() mapof.Any {
	return mapof.Any{
		"templateId":      "test-collection",
		"label":           "Notes",
		"data.layout":     "notes",
		"groups.viewer.0": "000000000000000000000000",
	}
}

/******************************************
 * newDefaultChild
 ******************************************/

// TestNewDefaultChild_PlacesTheStreamInTheOutbox confirms that an entry's values are applied, and
// that the Stream lands in the User's outbox with the User as its author
func TestNewDefaultChild_PlacesTheStreamInTheOutbox(t *testing.T) {

	service := newDefaultChildService()
	user := newDefaultChildUser()

	stream, err := service.newDefaultChild(articlesEntry(), &user)
	require.NoError(t, err)

	// The entry's values
	require.Equal(t, "Articles", stream.Label)
	require.Equal(t, "column-3", stream.Data["layout"])
	require.EqualValues(t, 11, stream.Data["zoom"], "a quoted number is converted by the Template's schema")
	require.Equal(t, "everyone", stream.ReplyPolicy)
	require.Equal(t, id.Slice{model.MagicGroupIDAnonymous}, stream.Groups["viewer"])

	// The User's outbox
	require.Equal(t, "test-collection", stream.TemplateID)
	require.Equal(t, user.UserID, stream.ParentID)
	require.Equal(t, "profile", stream.NavigationID)
	require.Empty(t, stream.ParentIDs)
	require.Empty(t, stream.ParentTemplateID)

	// The User as author
	require.Equal(t, user.UserID, stream.AttributedTo.UserID)
	require.Equal(t, user.ProfileURL, stream.AttributedTo.ProfileURL)
	require.True(t, stream.IsAuthor(user.UserID))

	// Not published, the same as a collection that the User adds themselves
	require.Equal(t, model.NewStream().PublishDate, stream.PublishDate)
}

// TestNewDefaultChild_EntryCannotMoveTheStream confirms that the location and author always come
// from the User, even when an entry sets the same fields itself
func TestNewDefaultChild_EntryCannotMoveTheStream(t *testing.T) {

	service := newDefaultChildService()
	user := newDefaultChildUser()
	elsewhere := primitive.NewObjectID()

	entry := articlesEntry()
	entry["parentId"] = elsewhere.Hex()
	entry["navigationId"] = "elsewhere"
	entry["parentTemplateId"] = "test-collection"
	entry["attributedTo.userId"] = elsewhere.Hex()
	entry["attributedTo.name"] = "Someone Else"

	stream, err := service.newDefaultChild(entry, &user)
	require.NoError(t, err)

	require.Equal(t, user.UserID, stream.ParentID)
	require.Equal(t, "profile", stream.NavigationID)
	require.Empty(t, stream.ParentTemplateID)
	require.Equal(t, user.UserID, stream.AttributedTo.UserID)
	require.Equal(t, "Sarah Connor", stream.AttributedTo.Name)
}

// TestNewDefaultChild_RefusesAnUndefinedTemplate confirms that an entry naming no loaded Template is an error
func TestNewDefaultChild_RefusesAnUndefinedTemplate(t *testing.T) {

	service := newDefaultChildService()
	user := newDefaultChildUser()

	for _, templateID := range []string{"no-such-template", ""} {
		entry := articlesEntry()
		entry["templateId"] = templateID

		_, err := service.newDefaultChild(entry, &user)
		require.Error(t, err, "templateId %q", templateID)
		require.True(t, derp.IsNotFound(err), "templateId %q", templateID)
	}
}

// TestNewDefaultChild_RefusesATemplateOutsideTheOutbox confirms that a Template an outbox cannot hold is an error
func TestNewDefaultChild_RefusesATemplateOutsideTheOutbox(t *testing.T) {

	service := newDefaultChildService()
	user := newDefaultChildUser()

	_, err := service.newDefaultChild(mapof.Any{"templateId": "test-page", "label": "Page"}, &user)
	require.Error(t, err)
	require.Equal(t, 400, derp.ErrorCode(err))
}

// TestNewDefaultChild_RefusesAValueTheSchemaRefuses confirms that the named Template's own schema checks
// every value, including the "data" values that the base Stream schema would accept
func TestNewDefaultChild_RefusesAValueTheSchemaRefuses(t *testing.T) {

	service := newDefaultChildService()
	user := newDefaultChildUser()

	refused := map[string]mapof.Any{
		"a layout not in the enum":    {"data.layout": "notess"},
		"a zoom that is not a number": {"data.zoom": "eleven"},
		"a misspelled property":       {"lable": "Articles"},
		"a reply policy not in enum":  {"replyPolicy": "sometimes"},
		"a group that is not an ID":   {"groups.viewer.0": "anonymous"},
	}

	for name, values := range refused {
		entry := articlesEntry()

		for key, value := range values {
			entry[key] = value
		}

		_, err := service.newDefaultChild(entry, &user)
		require.Error(t, err, name)
	}
}

// TestNewDefaultChild_LeavesTheTemplateUnchanged confirms that building Streams for two Users at once
// gives each one its own Stream, and leaves the cached Template's entries exactly as they were
func TestNewDefaultChild_LeavesTheTemplateUnchanged(t *testing.T) {

	service := newDefaultChildService(articlesEntry(), notesEntry())
	outboxTemplate, err := service.templateService.Load("test-outbox")
	require.NoError(t, err)

	users := []model.User{newDefaultChildUser(), newDefaultChildUser()}
	results := make([][]model.Stream, len(users))
	errs := make([]error, len(users))

	var wait sync.WaitGroup

	for index := range users {
		wait.Add(1)

		go func() {
			defer wait.Done()

			for _, entry := range outboxTemplate.DefaultChildren {
				stream, err := service.newDefaultChild(entry, &users[index])

				if err != nil {
					errs[index] = err
					return
				}

				results[index] = append(results[index], stream)
			}
		}()
	}

	wait.Wait()

	for index, user := range users {
		require.NoError(t, errs[index])
		require.Len(t, results[index], 2)

		for _, stream := range results[index] {
			require.Equal(t, user.UserID, stream.ParentID)
		}
	}

	// Each User's Streams are their own
	require.NotEqual(t, results[0][0].StreamID, results[1][0].StreamID)
	results[0][0].Data["layout"] = "changed"
	results[0][0].Groups["viewer"] = id.Slice{}
	require.Equal(t, "column-3", results[1][0].Data["layout"])
	require.Equal(t, id.Slice{model.MagicGroupIDAnonymous}, results[1][0].Groups["viewer"])

	// The cached Template still holds the entries it was loaded with
	reloaded, err := service.templateService.Load("test-outbox")
	require.NoError(t, err)
	require.Equal(t, []mapof.Any{articlesEntry(), notesEntry()}, reloaded.DefaultChildren)
}

/******************************************
 * CreateDefaultChildren
 *
 * Its wrapped error for a failed outbox Template load is not covered,
 * because Template.Load fails only with a not-found error, which returns nil.
 ******************************************/

// TestCreateDefaultChildren_MissingOutboxTemplateCreatesNothing confirms that a User whose outbox Template is
// not loaded gets no Streams and no error.  The nil session proves that nothing reached the database.
func TestCreateDefaultChildren_MissingOutboxTemplateCreatesNothing(t *testing.T) {

	service := newDefaultChildService(articlesEntry())
	user := newDefaultChildUser()
	user.OutboxTemplate = "no-such-outbox"

	require.NoError(t, service.CreateDefaultChildren(nil, &user))
}

// TestCreateDefaultChildren_NoChildrenCreatesNothing confirms that an outbox Template with no default
// children creates nothing.  The nil session proves that nothing reached the database.
func TestCreateDefaultChildren_NoChildrenCreatesNothing(t *testing.T) {

	service := newDefaultChildService()
	user := newDefaultChildUser()

	require.NoError(t, service.CreateDefaultChildren(nil, &user))
}

// TestCreateDefaultChildren_ReachesSave is the inverse: an outbox Template with children must reach Save,
// which refuses the nil session
func TestCreateDefaultChildren_ReachesSave(t *testing.T) {

	service := newDefaultChildService(articlesEntry())
	user := newDefaultChildUser()

	err := service.CreateDefaultChildren(nil, &user)
	require.Error(t, err)
	require.Equal(t, "Session cannot be nil. This should never happen.", derp.Message(derp.RootCause(err)))
}

// TestCreateDefaultChildren_StopsAtABadEntry confirms that a bad entry is an error before anything is saved
func TestCreateDefaultChildren_StopsAtABadEntry(t *testing.T) {

	service := newDefaultChildService(mapof.Any{"templateId": "test-page", "label": "Page"}, articlesEntry())
	user := newDefaultChildUser()

	// A Save would refuse the nil session instead, so this error proves that none was attempted
	err := service.CreateDefaultChildren(nil, &user)
	require.Error(t, err)
	require.Equal(t, "Template cannot be contained by 'outbox'", derp.Message(derp.RootCause(err)))
	require.Equal(t, 400, derp.ErrorCode(err))
}

// TestCreateDefaultChildren_SavesEveryChild runs CreateDefaultChildren against a real MongoDB, and confirms
// that every child is stored in the User's outbox, in order, and shared with the Groups its entry names
func TestCreateDefaultChildren_SavesEveryChild(t *testing.T) {

	_, session := newReplicaSetSession(t)
	webhooks := &atomic.Int32{}

	service := newDefaultChildService(articlesEntry(), notesEntry())
	service.webhookService = &Webhook{
		newSession: func(time.Duration) (data.Session, context.CancelFunc, error) {
			webhooks.Add(1)
			return nil, nil, derp.Internal("test", "Webhooks are not delivered in this test")
		},
	}
	service.sseUpdateChannel = make(chan realtime.Message, 100)

	user := newDefaultChildUser()
	require.NoError(t, service.CreateDefaultChildren(session, &user))

	// Wait for both stream:create webhooks, so their goroutines do not outlive the test
	requireCount(t, webhooks, 2, "stream:create webhooks")

	streams := make([]model.Stream, 0)
	require.NoError(t, session.Collection("Stream").Query(&streams, exp.Equal("parentId", user.UserID), option.SortAsc("rank")))
	require.Len(t, streams, 2)

	// The declared order is the tab order
	require.Equal(t, "Articles", streams[0].Label)
	require.Equal(t, "Notes", streams[1].Label)
	require.Equal(t, 1, streams[0].Rank)
	require.Equal(t, 2, streams[1].Rank)

	for _, stream := range streams {
		require.Equal(t, "test-collection", stream.TemplateID)
		require.Equal(t, "default", stream.StateID)
		require.Equal(t, user.UserID, stream.AttributedTo.UserID)
		require.Equal(t, "https://example.test/"+stream.StreamID.Hex(), stream.URL)
		require.Contains(t, stream.DefaultAllow, model.MagicGroupIDAnonymous, "%s must be visible to everyone", stream.Label)
	}
}
