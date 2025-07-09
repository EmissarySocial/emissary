package build

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	builder "github.com/benpate/exp-builder"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Outbox builds individual messages from a User's Outbox.
type Outbox struct {
	_user *model.User
	CommonWithTemplate
}

// NewOutbox returns a fully initialized `Outbox` builder.
func NewOutbox(factory Factory, request *http.Request, response http.ResponseWriter, user *model.User, actionID string) (Outbox, error) {

	const location = "build.NewOutbox"

	// Load the Template
	templateService := factory.Template()
	template, err := templateService.Load(user.OutboxTemplate) // Users should get to choose their own outbox template

	if err != nil {
		return Outbox{}, derp.Wrap(err, location, "Error loading template")
	}

	// Create the underlying Common builder
	common, err := NewCommonWithTemplate(factory, request, response, template, user, actionID)

	if err != nil {
		return Outbox{}, derp.Wrap(err, location, "Error creating common builder")
	}

	// Verify that the User's profile is visible
	if !isUserVisible(&common._authorization, user) {
		return Outbox{}, derp.NotFoundError(location, "User not found")
	}

	// Enforce user permissions on the requested action
	if !common.UserCan(actionID) {
		if common._authorization.IsAuthenticated() {
			return Outbox{}, derp.ForbiddenError(location, "Forbidden")
		} else {
			return Outbox{}, derp.UnauthorizedError(location, "Anonymous user is not authorized to perform this action", user.ProfileURL, actionID)
		}
	}

	// Return the Outbox builder
	return Outbox{
		_user:              user,
		CommonWithTemplate: common,
	}, nil
}

/******************************************
 * Renderer Interface
 ******************************************/

// Render generates the string value for this Outbox
func (w Outbox) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w._action.Steps).Get(w._factory, &w, &buffer)

	if status.Error != nil {
		err := derp.Wrap(status.Error, "build.Outbox.Render", "Error generating HTML", w._request.URL.String())
		derp.Report(err)
		return "", err
	}

	// Success!
	status.Apply(w._response)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Outbox
func (w Outbox) View(actionID string) (template.HTML, error) {

	builder, err := NewOutbox(w._factory, w._request, w._response, w._user, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, "build.Outbox.View", "Error creating Outbox builder")
	}

	return builder.Render()
}

// NavigationID returns the ID to use for highlighing navigation menus
func (w Outbox) NavigationID() string {
	if w._user.UserID == w.AuthenticatedID() {
		return "outbox"
	}
	return "user"
}

func (w Outbox) PageTitle() string {
	return w._user.DisplayName
}

func (w Outbox) Permalink() string {
	return w.Host() + "/@" + w._user.UserID.Hex()
}

func (w Outbox) BasePath() string {
	return "/@" + w._user.UserID.Hex()
}

func (w Outbox) Token() string {
	return "users"
}

func (w Outbox) object() data.Object {
	return w._user
}

func (w Outbox) objectID() primitive.ObjectID {
	return w._user.UserID
}

func (w Outbox) objectType() string {
	return "User"
}

func (w Outbox) schema() schema.Schema {
	return schema.New(model.UserSchema())
}

func (w Outbox) service() service.ModelService {
	return w._factory.User()
}

func (w Outbox) templateRole() string {
	return "outbox"
}

func (w Outbox) clone(action string) (Builder, error) {
	return NewOutbox(w._factory, w._request, w._response, w._user, action)
}

// IsMyself returns TRUE if the outbox record is owned
// by the currently signed-in user
func (w Outbox) IsMyself() bool {
	return w._user.IsMyself(w._authorization.UserID)
}

/******************************************
 * Data Accessors
 ******************************************/

func (w Outbox) UserID() string {
	return w._user.UserID.Hex()
}

// Myself returns TRUE if the current user is viewing their own profile
func (w Outbox) Myself() bool {
	return w._user.IsMyself(w._authorization.UserID)
}

func (w Outbox) Username() string {
	return w._user.Username
}

func (w Outbox) RuleCount() int {
	return w._user.RuleCount
}

func (w Outbox) FollowerCount() int {
	return w._user.FollowerCount
}

func (w Outbox) FollowingCount() int {
	return w._user.FollowingCount
}

func (w Outbox) DisplayName() string {
	return w._user.DisplayName
}

func (w Outbox) StateID() string {
	return w._user.StateID
}

// IsPublished returns TRUE if the stream has been published
func (w Outbox) IsPublic() bool {
	return w._user.IsPublic
}

// IsIndexable returns TRUE if the stream is indexable by search engines
func (w Outbox) IsIndexable() bool {
	return w._user.IsIndexable
}

func (w Outbox) StatusMessage() string {
	return w._user.StatusMessage
}

func (w Outbox) ProfileURL() string {
	return w._user.ProfileURL
}

func (w Outbox) IconURL() string {
	return w._user.ActivityPubIconURL()
}

func (w Outbox) ImageURL() string {
	return w._user.ActivityPubImageURL()
}

func (w Outbox) Location() string {
	return w._user.Location
}

func (w Outbox) Links() sliceof.Object[model.PersonLink] {
	return w._user.Links
}

// Tags returns all tags (mentions, hashtags, etc) for the stream being built
func (w Outbox) Tags() sliceof.Object[mapof.String] {
	return slice.Map(w._user.Hashtags, func(tag string) mapof.String {
		return mapof.String{
			"Name": tag,
			"Type": vocab.LinkTypeHashtag,
			"Href": w.Host() + "/users?q=%23" + tag,
		}
	})
}

func (w Outbox) Data(path string) any {
	return w._user.Data[path]
}

// OEmbedJSON returns the URL for the oEmbed JSON endpoint for this stream
func (w Outbox) OEmbedJSON() string {
	return w.Host() + "/.oembed?url=" + w.Permalink() + "&format=json"
}

// OEmbedXML returns the URL for the oEmbed XML endpoint for this stream
func (w Outbox) OEmbedXML() string {
	return w.Host() + "/.oembed?url=" + w.Permalink() + "&format=xml"
}

func (w Outbox) ActivityPubURL() string {
	return w._user.ActivityPubURL()
}

func (w Outbox) ActivityPubIconURL() string {
	return w._user.ActivityPubIconURL()
}

func (w Outbox) ActivityPubInboxURL() string {
	return w._user.ActivityPubInboxURL()
}

func (w Outbox) ActivityPubOutboxURL() string {
	return w._user.ActivityPubOutboxURL()
}

func (w Outbox) ActivityPubFollowersURL() string {
	return w._user.ActivityPubFollowersURL()
}

func (w Outbox) ActivityPubFollowingURL() string {
	return w._user.ActivityPubFollowingURL()
}

func (w Outbox) ActivityPubLikedURL() string {
	return w._user.ActivityPubLikedURL()
}

func (w Outbox) ActivityPubPublicKeyURL() string {
	return w._user.ActivityPubPublicKeyURL()
}

/******************************************
 * Outbox Methods
 ******************************************/

func (w Outbox) Outbox() QueryBuilder[model.StreamSummary] {

	expressionBuilder := builder.NewBuilder().
		Int("publishDate")

	criteria := exp.And(
		expressionBuilder.Evaluate(w._request.URL.Query()),
		exp.Equal("parentId", w._user.UserID),
		exp.Equal("inReplyTo", ""),
		w.defaultAllowed(),
	)

	result := NewQueryBuilder[model.StreamSummary](w._factory.Stream(), criteria)

	return result
}

func (w Outbox) Circles() QueryBuilder[model.Circle] {

	expressionBuilder := builder.NewBuilder().
		String("name")

	criteria := exp.And(
		expressionBuilder.Evaluate(w._request.URL.Query()),
		exp.Equal("userId", w.objectID()),
	)

	result := NewQueryBuilder[model.Circle](w._factory.Circle(), criteria)

	return result
}

func (w Outbox) HasProducts() (bool, error) {
	return w._factory.Circle().HasProducts(w._user.UserID)
}

func (w Outbox) ProductCount() (int, error) {
	return w._factory.Circle().ProductCount(w._user.UserID)
}

func (w Outbox) Products() (sliceof.Object[model.Product], error) {

	const location = "build.Outbox.Products"

	// Get purchaseable products from all Featured Circles
	productIDs, err := w._factory.Circle().AssignedProductIDs(w._user.UserID)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error retrieving remote products for user", w._user.UserID.Hex())
	}

	// If there are no remote products, return an empty slice
	if productIDs.IsEmpty() {
		return sliceof.Object[model.Product]{}, nil
	}

	// Look up the products for this User using their IDs
	products, err := w._factory.Product().QueryByIDs(w._user.UserID, productIDs...)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error retrieving remote products for user", w._user.UserID.Hex())
	}

	return products, nil
}

func (w Outbox) Replies() QueryBuilder[model.StreamSummary] {

	expressionBuilder := builder.NewBuilder().
		Int("publishDate")

	criteria := exp.And(
		expressionBuilder.Evaluate(w._request.URL.Query()),
		exp.Equal("parentId", w._user.UserID),
		exp.NotEqual("inReplyTo", ""),
	)

	result := NewQueryBuilder[model.StreamSummary](w._factory.Stream(), criteria)

	return result
}

func (w Outbox) Responses() QueryBuilder[model.Response] {

	expressionBuilder := builder.NewBuilder().
		Int("createDate")

	criteria := exp.And(
		expressionBuilder.Evaluate(w._request.URL.Query()),
		exp.Equal("userId", w.objectID()),
	)

	result := NewQueryBuilder[model.Response](w._factory.Response(), criteria)

	return result
}

func (w Outbox) setState(stateID string) error {
	w._user.SetState(stateID)
	return nil
}

func (w Outbox) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_Outbox")
}
