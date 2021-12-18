package render

import (
	"bytes"
	"html/template"
	"io"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/exp/builder"
	"github.com/benpate/ghost/model"
	"github.com/benpate/path"
	"github.com/benpate/schema"
	"github.com/benpate/steranko"
)

type Content struct {
	user     model.User
	layout   model.Layout
	actionID string
	Common
}

func NewContent(factory Factory, ctx *steranko.Context, user model.User, actionID string) User {

	layoutService := factory.Layout()
	layout := layoutService.User()

	return User{
		user:     user,
		layout:   layout,
		actionID: actionID,
		Common:   NewCommon(factory, ctx),
	}
}

/*******************************************
 * PATH INTERFACE
 * (not available via templates)
 *******************************************/

func (content *Content) GetPath(p path.Path) (interface{}, error) {
	return content.user.GetPath(p)
}

func (content *Content) SetPath(p path.Path, value interface{}) error {
	return content.user.SetPath(p, value)
}

/*******************************************
 * RENDERER INTERFACE
 *******************************************/

// ActionID returns the unique ID of the Action configured into this renderer
func (content Content) ActionID() string {
	return content.actionID
}

// Action returns the model.Action configured into this renderer
func (content Content) Action() (model.Action, bool) {
	return content.layout.Action(content.ActionID())
}

// Render generates the string value for this Stream
func (content Content) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	if action, ok := content.layout.Action(content.actionID); ok {

		// Execute step (write HTML to buffer, update context)
		if err := DoPipeline(content.factory, &content, &buffer, action.Steps, ActionMethodGet); err != nil {
			return "", derp.Report(derp.Wrap(err, "ghost.render.Stream.Render", "Error generating HTML"))
		}
	}

	// Success!
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this User
func (content Content) View(actionID string) (template.HTML, error) {
	return NewUser(content.factory, content.ctx, content.user, actionID).Render()
}

func (content Content) TopLevelID() string {
	return "admin"
}

func (content Content) Token() string {
	return "content"
}

func (content Content) object() data.Object {
	return &content.user
}

func (content Content) schema() schema.Schema {
	return content.user.Schema()
}

func (content Content) common() Common {
	return content.Common
}

func (content Content) executeTemplate(writer io.Writer, name string, data interface{}) error {
	return content.layout.HTMLTemplate.ExecuteTemplate(writer, name, data)
}

/*******************************************
 * DATA ACCESSORS
 *******************************************/

func (content Content) UserID() string {
	return content.user.UserID.Hex()
}

func (content Content) DisplayName() string {
	return content.user.DisplayName
}

func (content Content) AvatarURL() string {
	return content.user.AvatarURL
}

/*******************************************
 * QUERY BUILDERS
 *******************************************/

func (content Content) Users() *QueryBuilder {

	query := builder.NewBuilder().
		String("displayName").
		ObjectID("groupId")

	criteria := exp.And(
		query.Evaluate(content.ctx.Request().URL.Query()),
		exp.Equal("journal.deleteDate", 0),
	)

	result := NewQueryBuilder(content.factory, content.ctx, content.factory.User(), criteria)

	return &result
}

/*******************************************
 * ADDITIONAL DATA
 *******************************************/

// AdminSections returns labels and values for all hard-coded sections of the administrator area.
func (content Content) AdminSections() []model.Option {
	return AdminSections()
}
