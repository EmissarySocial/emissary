package render

import (
	"bytes"
	"html/template"
	"io"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/exp/builder"
	"github.com/benpate/schema"
	"github.com/benpate/steranko"
	"github.com/whisperverse/whisperverse/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Profile struct {
	layout *model.Layout
	action *model.Action
	user   *model.User
	Common
}

func NewProfile(factory Factory, ctx *steranko.Context, layout *model.Layout, action *model.Action, user *model.User) Profile {

	return Profile{
		layout: layout,
		action: action,
		user:   user,
		Common: NewCommon(factory, ctx),
	}
}

/*******************************************
 * RENDERER INTERFACE
 *******************************************/

// ActionID returns the unique ID of the Action configured into this renderer
func (w Profile) ActionID() string {
	return w.action.ActionID
}

// Action returns the model.Action configured into this renderer
func (w Profile) Action() *model.Action {
	return w.action
}

// Render generates the string value for this Profile
func (w Profile) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	if err := DoPipeline(&w, &buffer, w.action.Steps, ActionMethodGet); err != nil {
		return "", derp.Report(derp.Wrap(err, "whisper.render.Profile.Render", "Error generating HTML"))

	}

	// Success!
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Profile
func (w Profile) View(actionID string) (template.HTML, error) {

	action := w.layout.Action(actionID)

	return NewProfile(w.factory(), w.ctx, w.layout, action, w.user).Render()
}

func (w Profile) TopLevelID() string {
	return ""
}

func (w Profile) Token() string {
	return "users"
}

func (w Profile) object() data.Object {
	return w.user
}

func (w Profile) objectID() primitive.ObjectID {
	return w.user.UserID
}

func (w Profile) schema() schema.Schema {
	return w.user.Schema()
}

func (w Profile) service() ModelService {
	return w.f.User()
}

func (w Profile) executeTemplate(writer io.Writer, name string, data interface{}) error {
	return w.layout.HTMLTemplate.ExecuteTemplate(writer, name, data)
}

/*******************************************
 * DATA ACCESSORS
 *******************************************/

func (w Profile) UserID() string {
	return w.user.UserID.Hex()
}

func (w Profile) DisplayName() string {
	return w.user.DisplayName
}

func (w Profile) AvatarURL() string {
	return w.user.AvatarURL
}

/*******************************************
 * QUERY BUILDERS
 *******************************************/

func (w Profile) Profiles() *QueryBuilder {

	query := builder.NewBuilder().
		String("displayName").
		ObjectID("groupId")

	criteria := exp.And(
		query.Evaluate(w.ctx.Request().URL.Query()),
		exp.Equal("journal.deleteDate", 0),
	)

	result := NewQueryBuilder(w.factory(), w.ctx, w.factory().User(), criteria)

	return &result
}
