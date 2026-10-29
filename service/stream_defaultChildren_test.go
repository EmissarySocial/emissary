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

// newDefaultChildTemplates returns the Templates these tests share: an outbox that declares the given
// children, a collection that an outbox can hold, an item that a collection can hold, and a page that neither can
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

	item := model.NewTemplate("test-item", nil)
	item.TemplateRole = "test-item"
	item.Model = "Stream"
	item.ContainedBy = sliceof.String{"test-collection"}
	item.States = mapof.Object[model.State]{"default": {StateID: "default"}}
	item.AccessRoles = mapof.Object[model.Role]{"viewer": {RoleID: "viewer"}}
	item.Actions = mapof.Object[model.Action]{"view": {
		Roles:      sliceof.String{"author", "viewer"},
		AccessList: mapof.Object[sliceof.String]{"default": {"author", "viewer"}},
	}}
	item.Schema = schema.New(schema.Object{Properties: schema.ElementMap{
		"data": schema.Object{Properties: schema.ElementMap{
			"size": schema.String{Enum: []string{"small", "large"}},
		}},
	}})
	item.Schema.Inherit(schema.New(item.BaseSchema()))

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
		item.TemplateID:       item,
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
 * newDefaultChildForUser
 ******************************************/

// TestNewDefaultChildForUser_PlacesTheStreamInTheOutbox confirms that an entry's values are applied, and
// that the Stream lands in the User's outbox with the User as its author
func TestNewDefaultChildForUser_PlacesTheStreamInTheOutbox(t *testing.T) {

	service := newDefaultChildService()
	user := newDefaultChildUser()

	stream, err := service.newDefaultChildForUser(articlesEntry(), &user)
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

// TestNewDefaultChildForUser_EntryCannotMoveTheStream confirms that the location and author always come
// from the User, even when an entry sets the same fields itself
func TestNewDefaultChildForUser_EntryCannotMoveTheStream(t *testing.T) {

	service := newDefaultChildService()
	user := newDefaultChildUser()
	elsewhere := primitive.NewObjectID()

	entry := articlesEntry()
	entry["parentId"] = elsewhere.Hex()
	entry["navigationId"] = "elsewhere"
	entry["parentTemplateId"] = "test-collection"
	entry["attributedTo.userId"] = elsewhere.Hex()
	entry["attributedTo.name"] = "Someone Else"

	stream, err := service.newDefaultChildForUser(entry, &user)
	require.NoError(t, err)

	require.Equal(t, user.UserID, stream.ParentID)
	require.Equal(t, "profile", stream.NavigationID)
	require.Empty(t, stream.ParentTemplateID)
	require.Equal(t, user.UserID, stream.AttributedTo.UserID)
	require.Equal(t, "Sarah Connor", stream.AttributedTo.Name)
}

// TestNewDefaultChildForUser_RefusesAnUndefinedTemplate confirms that an entry naming no loaded Template is an error
func TestNewDefaultChildForUser_RefusesAnUndefinedTemplate(t *testing.T) {

	service := newDefaultChildService()
	user := newDefaultChildUser()

	for _, templateID := range []string{"no-such-template", ""} {
		entry := articlesEntry()
		entry["templateId"] = templateID

		_, err := service.newDefaultChildForUser(entry, &user)
		require.Error(t, err, "templateId %q", templateID)
		require.True(t, derp.IsNotFound(err), "templateId %q", templateID)
	}
}

// TestNewDefaultChildForUser_RefusesATemplateOutsideTheOutbox confirms that a Template an outbox cannot hold is an error
func TestNewDefaultChildForUser_RefusesATemplateOutsideTheOutbox(t *testing.T) {

	service := newDefaultChildService()
	user := newDefaultChildUser()

	_, err := service.newDefaultChildForUser(mapof.Any{"templateId": "test-page", "label": "Page"}, &user)
	require.Error(t, err)
	require.Equal(t, 400, derp.ErrorCode(err))
}

// TestNewDefaultChildForUser_RefusesAValueTheSchemaRefuses confirms that the named Template's own schema checks
// every value, including the "data" values that the base Stream schema would accept
func TestNewDefaultChildForUser_RefusesAValueTheSchemaRefuses(t *testing.T) {

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

		_, err := service.newDefaultChildForUser(entry, &user)
		require.Error(t, err, name)
	}
}

// TestNewDefaultChildForUser_LeavesTheTemplateUnchanged confirms that building Streams for two Users at once
// gives each one its own Stream, and leaves the cached Template's entries exactly as they were
func TestNewDefaultChildForUser_LeavesTheTemplateUnchanged(t *testing.T) {

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
				stream, err := service.newDefaultChildForUser(entry, &users[index])

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
 * CreateDefaultChildrenForUser
 *
 * Its wrapped error for a failed outbox Template load is not covered,
 * because Template.Load fails only with a not-found error, which returns nil.
 ******************************************/

// TestCreateDefaultChildrenForUser_MissingOutboxTemplateCreatesNothing confirms that a User whose outbox Template is
// not loaded gets no Streams and no error.  The nil session proves that nothing reached the database.
func TestCreateDefaultChildrenForUser_MissingOutboxTemplateCreatesNothing(t *testing.T) {

	service := newDefaultChildService(articlesEntry())
	user := newDefaultChildUser()
	user.OutboxTemplate = "no-such-outbox"

	require.NoError(t, service.CreateDefaultChildrenForUser(nil, &user))
}

// TestCreateDefaultChildrenForUser_NoChildrenCreatesNothing confirms that an outbox Template with no default
// children creates nothing.  The nil session proves that nothing reached the database.
func TestCreateDefaultChildrenForUser_NoChildrenCreatesNothing(t *testing.T) {

	service := newDefaultChildService()
	user := newDefaultChildUser()

	require.NoError(t, service.CreateDefaultChildrenForUser(nil, &user))
}

// TestCreateDefaultChildrenForUser_ReachesSave is the inverse: an outbox Template with children must reach Save,
// which refuses the nil session
func TestCreateDefaultChildrenForUser_ReachesSave(t *testing.T) {

	service := newDefaultChildService(articlesEntry())
	user := newDefaultChildUser()

	err := service.CreateDefaultChildrenForUser(nil, &user)
	require.Error(t, err)
	require.Equal(t, "Session cannot be nil. This should never happen.", derp.Message(derp.RootCause(err)))
}

// TestCreateDefaultChildrenForUser_StopsAtABadEntry confirms that a bad entry is an error before anything is saved
func TestCreateDefaultChildrenForUser_StopsAtABadEntry(t *testing.T) {

	service := newDefaultChildService(mapof.Any{"templateId": "test-page", "label": "Page"}, articlesEntry())
	user := newDefaultChildUser()

	// A Save would refuse the nil session instead, so this error proves that none was attempted
	err := service.CreateDefaultChildrenForUser(nil, &user)
	require.Error(t, err)
	require.Equal(t, "Template cannot be contained by 'outbox'", derp.Message(derp.RootCause(err)))
	require.Equal(t, 400, derp.ErrorCode(err))
}

// TestCreateDefaultChildrenForUser_SavesEveryChild runs CreateDefaultChildren against a real MongoDB, and confirms
// that every child is stored in the User's outbox, in order, and shared with the Groups its entry names
func TestCreateDefaultChildrenForUser_SavesEveryChild(t *testing.T) {

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
	require.NoError(t, service.CreateDefaultChildrenForUser(session, &user))

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

/******************************************
 * Default children of a Stream
 ******************************************/

// newDefaultChildStreamService returns a Stream service whose collection Template declares the given children
func newDefaultChildStreamService(entries ...mapof.Any) *Stream {

	templates := newDefaultChildTemplates()

	collection := templates["test-collection"]
	collection.DefaultChildren = entries
	templates["test-collection"] = collection

	return &Stream{
		host:            "https://example.test",
		templateService: &Template{templates: templates},
	}
}

// newDefaultChildParent returns a collection in the User's outbox that has already been saved
func newDefaultChildParent(user model.User) model.Stream {
	parent := model.NewStream()
	parent.TemplateID = "test-collection"
	parent.ParentID = user.UserID
	parent.ParentIDs = []primitive.ObjectID{user.UserID}
	parent.NavigationID = "profile"
	parent.AttributedTo = user.PersonLink()
	parent.CreateDate = time.Now().UnixMilli()
	return parent
}

// itemEntry returns a default child that a collection can hold
func itemEntry(label string) mapof.Any {
	return mapof.Any{"templateId": "test-item", "label": label, "data.size": "small"}
}

// TestNewDefaultChildForStream_PlacesTheStreamBeneathItsParent confirms that an entry's values are applied, and
// that the Stream lands beneath the parent with the parent's author
func TestNewDefaultChildForStream_PlacesTheStreamBeneathItsParent(t *testing.T) {

	service := newDefaultChildStreamService()
	user := newDefaultChildUser()
	parent := newDefaultChildParent(user)

	stream, err := service.newDefaultChildForStream(itemEntry("First"), &parent)
	require.NoError(t, err)

	// The entry's values
	require.Equal(t, "First", stream.Label)
	require.Equal(t, "small", stream.Data["size"])

	// Beneath the parent
	require.Equal(t, "test-item", stream.TemplateID)
	require.Equal(t, parent.StreamID, stream.ParentID)
	require.Equal(t, []primitive.ObjectID{user.UserID, parent.StreamID}, []primitive.ObjectID(stream.ParentIDs))
	require.Equal(t, "profile", stream.NavigationID)
	require.Equal(t, "test-collection", stream.ParentTemplateID)

	// With the parent's author
	require.Equal(t, parent.AttributedTo, stream.AttributedTo)
	require.True(t, stream.IsAuthor(user.UserID))

	// Not published
	require.Equal(t, model.NewStream().PublishDate, stream.PublishDate)
}

// TestNewDefaultChildForStream_EntryCannotMoveTheStream confirms that the location and author always come
// from the parent, even when an entry sets the same fields itself
func TestNewDefaultChildForStream_EntryCannotMoveTheStream(t *testing.T) {

	service := newDefaultChildStreamService()
	user := newDefaultChildUser()
	parent := newDefaultChildParent(user)
	elsewhere := primitive.NewObjectID()

	entry := itemEntry("First")
	entry["parentId"] = elsewhere.Hex()
	entry["parentIds.0"] = elsewhere.Hex()
	entry["navigationId"] = "elsewhere"
	entry["parentTemplateId"] = "test-page"
	entry["attributedTo.userId"] = elsewhere.Hex()

	stream, err := service.newDefaultChildForStream(entry, &parent)
	require.NoError(t, err)

	require.Equal(t, parent.StreamID, stream.ParentID)
	require.Equal(t, []primitive.ObjectID{user.UserID, parent.StreamID}, []primitive.ObjectID(stream.ParentIDs))
	require.Equal(t, "profile", stream.NavigationID)
	require.Equal(t, "test-collection", stream.ParentTemplateID)
	require.Equal(t, user.UserID, stream.AttributedTo.UserID)
}

// TestNewDefaultChildForStream_RefusesATemplateItsParentCannotHold confirms that a collection cannot be made
// beneath a collection, because only an outbox can hold one
func TestNewDefaultChildForStream_RefusesATemplateItsParentCannotHold(t *testing.T) {

	service := newDefaultChildStreamService()
	parent := newDefaultChildParent(newDefaultChildUser())

	_, err := service.newDefaultChildForStream(articlesEntry(), &parent)
	require.Error(t, err)
	require.Equal(t, 400, derp.ErrorCode(err))
}

// TestNewDefaultChildForStream_RefusesAValueTheSchemaRefuses confirms that the item Template's own schema checks the values
func TestNewDefaultChildForStream_RefusesAValueTheSchemaRefuses(t *testing.T) {

	service := newDefaultChildStreamService()
	parent := newDefaultChildParent(newDefaultChildUser())

	entry := itemEntry("First")
	entry["data.size"] = "medium"

	_, err := service.newDefaultChildForStream(entry, &parent)
	require.Error(t, err)
}

// TestCreateDefaultChildrenForStream_UnsavedParentCreatesNothing confirms that a Stream that was never saved, such
// as one whose "create" action halted before its save, gets no children.  The nil session proves that nothing reached the database.
func TestCreateDefaultChildrenForStream_UnsavedParentCreatesNothing(t *testing.T) {

	service := newDefaultChildStreamService(itemEntry("First"))
	parent := newDefaultChildParent(newDefaultChildUser())
	parent.CreateDate = 0

	require.NoError(t, service.CreateDefaultChildrenForStream(nil, &parent))
}

// TestCreateDefaultChildrenForStream_NoChildrenCreatesNothing confirms that a Template with no default children
// creates nothing.  The nil session proves that nothing reached the database.
func TestCreateDefaultChildrenForStream_NoChildrenCreatesNothing(t *testing.T) {

	service := newDefaultChildStreamService()
	parent := newDefaultChildParent(newDefaultChildUser())

	require.NoError(t, service.CreateDefaultChildrenForStream(nil, &parent))
}

// TestCreateDefaultChildrenForStream_ReachesSave is the inverse: a saved parent whose Template has children must
// reach Save, which refuses the nil session
func TestCreateDefaultChildrenForStream_ReachesSave(t *testing.T) {

	service := newDefaultChildStreamService(itemEntry("First"))
	parent := newDefaultChildParent(newDefaultChildUser())

	err := service.CreateDefaultChildrenForStream(nil, &parent)
	require.Error(t, err)
	require.Equal(t, "Session cannot be nil. This should never happen.", derp.Message(derp.RootCause(err)))
}

// TestCreateDefaultChildrenForStream_StopsAtABadEntry confirms that a bad entry is an error before anything is saved
func TestCreateDefaultChildrenForStream_StopsAtABadEntry(t *testing.T) {

	service := newDefaultChildStreamService(articlesEntry(), itemEntry("First"))
	parent := newDefaultChildParent(newDefaultChildUser())

	// A Save would refuse the nil session instead, so this error proves that none was attempted
	err := service.CreateDefaultChildrenForStream(nil, &parent)
	require.Error(t, err)
	require.Equal(t, "Template cannot be contained by parent", derp.Message(derp.RootCause(err)))
	require.Equal(t, 400, derp.ErrorCode(err))
}

// TestCreateDefaultChildrenForStream_MissingParentTemplate confirms that a parent whose Template is not loaded is an error
func TestCreateDefaultChildrenForStream_MissingParentTemplate(t *testing.T) {

	service := newDefaultChildStreamService(itemEntry("First"))
	parent := newDefaultChildParent(newDefaultChildUser())
	parent.TemplateID = "no-such-template"

	err := service.CreateDefaultChildrenForStream(nil, &parent)
	require.Error(t, err)
	require.True(t, derp.IsNotFound(err))
}

// TestCreateDefaultChildrenForStream_SavesEveryChild runs CreateDefaultChildrenForStream against a real MongoDB, and
// confirms that every child is stored beneath the parent, in order, with the parent's sharing, and with no children of its own
func TestCreateDefaultChildrenForStream_SavesEveryChild(t *testing.T) {

	_, session := newReplicaSetSession(t)
	webhooks := &atomic.Int32{}

	service := newDefaultChildStreamService(itemEntry("First"), itemEntry("Second"))
	service.webhookService = &Webhook{
		newSession: func(time.Duration) (data.Session, context.CancelFunc, error) {
			webhooks.Add(1)
			return nil, nil, derp.Internal("test", "Webhooks are not delivered in this test")
		},
	}
	service.sseUpdateChannel = make(chan realtime.Message, 100)

	// The collection shares with its children, and an item lists an item, which must not be created
	templates := service.templateService.templates

	collection := templates["test-collection"]
	collection.ShareWithChildren = true
	templates["test-collection"] = collection

	item := templates["test-item"]
	item.DefaultChildren = []mapof.Any{itemEntry("Grandchild")}
	templates["test-item"] = item

	// Save a public collection in the User's outbox, the way add-stream's "create" does
	user := newDefaultChildUser()
	parent := model.NewStream()
	require.NoError(t, service.SetLocationOutbox(&collection, &parent, user.UserID))
	parent.Label = "Collection"
	parent.AttributedTo = user.PersonLink()
	parent.Groups["viewer"] = id.Slice{model.MagicGroupIDAnonymous}
	require.NoError(t, service.Save(session, &parent, "test setup"))

	require.NoError(t, service.CreateDefaultChildrenForStream(session, &parent))

	// Wait for the three stream:create webhooks, so their goroutines do not outlive the test
	requireCount(t, webhooks, 3, "stream:create webhooks")

	children := make([]model.Stream, 0)
	require.NoError(t, session.Collection("Stream").Query(&children, exp.Equal("parentId", parent.StreamID), option.SortAsc("rank")))
	require.Len(t, children, 2)

	// The declared order is the rank order
	require.Equal(t, "First", children[0].Label)
	require.Equal(t, "Second", children[1].Label)
	require.Equal(t, 1, children[0].Rank)
	require.Equal(t, 2, children[1].Rank)

	for _, child := range children {
		require.Equal(t, "test-item", child.TemplateID)
		require.Equal(t, "test-collection", child.ParentTemplateID)
		require.Equal(t, []primitive.ObjectID{user.UserID, parent.StreamID}, []primitive.ObjectID(child.ParentIDs))
		require.Equal(t, user.UserID, child.AttributedTo.UserID)
		require.Equal(t, id.Slice{model.MagicGroupIDAnonymous}, child.Groups["viewer"], "%s must take the parent's sharing", child.Label)
		require.Contains(t, child.DefaultAllow, model.MagicGroupIDAnonymous, "%s must be visible to everyone", child.Label)

		// RULE: Default children get no default children of their own
		count, err := session.Collection("Stream").Count(exp.Equal("parentId", child.StreamID))
		require.NoError(t, err)
		require.Zero(t, count, "%s must have no children", child.Label)
	}
}
