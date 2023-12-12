package render

import (
	"bytes"
	"html/template"
	"io"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	builder "github.com/benpate/exp-builder"
	"github.com/benpate/rosetta/schema"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Block is a renderer for the admin/blocks page
// It can only be accessed by a Domain Owner
type Block struct {
	_block *model.Block
	Common
}

// NewBlock returns a fully initialized `Block` renderer.
func NewBlock(factory Factory, request *http.Request, response http.ResponseWriter, block *model.Block, template model.Template, actionID string) (Block, error) {

	const location = "render.NewBlock"

	// Create the underlying Common renderer
	common, err := NewCommon(factory, request, response, template, actionID)

	if err != nil {
		return Block{}, derp.Wrap(err, location, "Error creating common renderer")
	}

	// Verify that the user is a Domain Owner
	if !common._authorization.DomainOwner {
		return Block{}, derp.NewForbiddenError(location, "Must be domain owner to continue")
	}

	// Return the Block renderer
	return Block{
		_block: block,
		Common: common,
	}, nil
}

/******************************************
 * RENDERER INTERFACE
 ******************************************/

// Render generates the string value for this Stream
func (w Block) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w.action.Steps).Get(w._factory, &w, &buffer)

	if status.Error != nil {
		err := derp.Wrap(status.Error, "render.Block.Render", "Error generating HTML")
		derp.Report(err)
		return "", err
	}

	// Success!
	status.Apply(w._response)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Block
func (w Block) View(actionID string) (template.HTML, error) {

	const location = "render.Block.View"

	renderer, err := NewBlock(w._factory, w._request, w._response, w._block, w._template, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, location, "Error creating Block renderer")
	}

	return renderer.Render()
}

func (w Block) NavigationID() string {
	return "admin"
}

func (w Block) Permalink() string {
	return w.Hostname() + "/admin/blocks/" + w.BlockID()
}

func (w Block) BasePath() string {
	return "/admin/blocks/" + w.BlockID()
}

func (w Block) Token() string {
	return "blocks"
}

func (w Block) PageTitle() string {
	return "Settings"
}

func (w Block) object() data.Object {
	return w._block
}

func (w Block) objectID() primitive.ObjectID {
	return w._block.BlockID
}

func (w Block) objectType() string {
	return "Block"
}

func (w Block) schema() schema.Schema {
	return schema.New(model.BlockSchema())
}

func (w Block) service() service.ModelService {
	return w._factory.Block()
}

func (w Block) executeTemplate(writer io.Writer, name string, data any) error {
	return w._template.HTMLTemplate.ExecuteTemplate(writer, name, data)
}

func (w Block) clone(action string) (Renderer, error) {
	return NewBlock(w._factory, w._request, w._response, w._block, w._template, action)
}

/******************************************
 * DATA ACCESSORS
 ******************************************/

func (w Block) BlockID() string {
	if w._block == nil {
		return ""
	}
	return w._block.BlockID.Hex()
}

func (w Block) Label() string {
	if w._block == nil {
		return ""
	}
	return w._block.Label
}

/******************************************
 * QUERY BUILDERS
 ******************************************/

func (w Block) Blocks() *QueryBuilder[model.Block] {

	query := builder.NewBuilder().
		String("type").
		String("behavior").
		String("trigger")

	criteria := exp.And(
		query.Evaluate(w._request.URL.Query()),
		exp.Equal("userId", w._authorization.UserID),
		exp.Equal("deleteDate", 0),
	)

	result := NewQueryBuilder[model.Block](w._factory.Block(), criteria)

	return &result
}

func (w Block) ServerWideBlocks() *QueryBuilder[model.Block] {

	query := builder.NewBuilder().
		String("type").
		String("behavior").
		String("trigger")

	criteria := exp.And(
		query.Evaluate(w._request.URL.Query()),
		exp.Equal("userId", primitive.NilObjectID),
		exp.Equal("deleteDate", 0),
	)

	result := NewQueryBuilder[model.Block](w._factory.Block(), criteria)

	return &result
}

func (w Block) debug() {
	log.Debug().Interface("object", w.object()).Msg("renderer_admin_block")
}
