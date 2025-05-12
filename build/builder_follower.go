package build

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/schema"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Follower builds objects from any model service that implements the ModelService interface
type Follower struct {
	_follower *model.Follower
	CommonWithTemplate
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// NewFollower returns a fully initialized `Follower` builder.
func NewFollower(factory Factory, request *http.Request, response http.ResponseWriter, template model.Template, follower *model.Follower, actionID string) (Follower, error) {

	const location = "build.NewFollower"

	common, err := NewCommonWithTemplate(factory, request, response, template, actionID)

	if err != nil {
		return Follower{}, derp.Wrap(err, "build.NewFollower", "Error creating new model")
	}

	// Enforce user permissions on the requested action
	if !common._action.UserCan(follower, &common._authorization) {
		if common._authorization.IsAuthenticated() {
			return Follower{}, derp.ReportAndReturn(derp.NewForbiddenError(location, "Forbidden"))
		} else {
			return Follower{}, derp.ReportAndReturn(derp.NewUnauthorizedError(location, "Anonymous user is not authorized to perform this action", actionID))
		}
	}

	builder := Follower{
		_follower:          follower,
		CommonWithTemplate: common,
	}

	return builder, nil
}

/******************************************
 * Custom Methods for Follower builder
 ******************************************/

// AmFollowing returns a Following record for the current user and the given URL
// If the user is not authenticated, or the URL is not valid, then an empty Following record is returned.
// The UX uses this to label "mutual" follows
func (w Follower) AmFollowing(url string) model.Following {

	if !w._authorization.IsAuthenticated() {
		return model.NewFollowing()
	}

	// Get following service and new following record
	followingService := w._factory.Following()
	following := model.NewFollowing()

	// Retrieve following record. Discard errors
	// nolint:errcheck
	_ = followingService.LoadByURL(w._authorization.UserID, url, &following)

	// Return the (possibly empty) Following record
	return following
}

// FollowerID returns the FollowerID property of this Follow
func (w Follower) FollowerID() primitive.ObjectID {
	return w._follower.FollowerID
}

// ParentType returns the ParentType property of this Follow
func (w Follower) ParentType() string {
	return w._follower.ParentType
}

// ParentID returns the ParentID property of this Follow
func (w Follower) ParentID() primitive.ObjectID {
	return w._follower.ParentID
}

// ActorID returns the ActorID property of this Follow
func (w Follower) StateID() string {
	return w._follower.StateID
}

// Method returns the Method property of this Follow
func (w Follower) Method() string {
	return w._follower.Method
}

// Format returns the Format property of this Follow
func (w Follower) Format() string {
	return w._follower.Format
}

// ActorID returns the ActorID property of this Follow
func (w Follower) Actor() model.PersonLink {
	return w._follower.Actor
}

// Data returns a single data value from this Follow
func (w Follower) Data(value string) any {
	return w._follower.Data[value]
}

// CreateDate returns the CreateDate property of this Follow
func (w Follower) CreateDate() int64 {
	return w._follower.CreateDate
}

// UpdateDate returns the UpdateDate property of this Follow
func (w Follower) UpdateDate() int64 {
	return w._follower.UpdateDate
}

// ExpireDate returns the ExpireDate property of this Follow
func (w Follower) ExpireDate() int64 {
	return w._follower.ExpireDate
}

/******************************************
 * Builder Interface
 ******************************************/

func (w Follower) Follower() model.Follower {
	return *w._follower
}

func (w Follower) object() data.Object {
	return w._follower
}

func (w Follower) objectType() string {
	return "Follower"
}

func (w Follower) objectID() primitive.ObjectID {
	return w._follower.FollowerID
}

func (w Follower) schema() schema.Schema {
	return schema.New(model.FollowerSchema())
}

func (w Follower) service() service.ModelService {
	return w._factory.Follower()
}

func (w Follower) Label() string {
	return w._follower.Actor.Name
}

func (w Follower) Token() string {
	return ""
}

func (w Follower) PageTitle() string {
	return ""
}

func (w Follower) Permalink() string {
	return ""
}

func (w Follower) BasePath() string {
	return ""
}

func (w Follower) UserCan(string) bool {
	return false
}

func (w Follower) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w._action.Steps).Get(w._factory, &w, &buffer)

	if status.Error != nil {
		err := derp.Wrap(status.Error, "build.Follower.Render", "Error generating HTML")
		derp.Report(err)
		return "", err
	}

	// Success!
	status.Apply(w._response)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Stream
func (w Follower) View(actionID string) (template.HTML, error) {

	const location = "build.Follower.View"

	// Create a new builder (this will also validate the user's permissions)
	subStream, err := NewModel(w._factory, w._request, w._response, w._template, w._follower, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, location, "Error creating sub-builder")
	}

	// Generate HTML template
	return subStream.Render()
}

func (w Follower) setState(stateID string) error {
	return nil
}

func (w Follower) clone(action string) (Builder, error) {
	return NewFollower(w._factory, w._request, w._response, w._template, w._follower, action)
}

func (w Follower) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_Model")
}
