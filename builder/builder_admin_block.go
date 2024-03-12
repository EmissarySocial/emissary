package builder

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

// Rule is a builder for the admin/rules page
// It can only be accessed by a Domain Owner
type Rule struct {
	_rule *model.Rule
	Common
}

// NewRule returns a fully initialized `Rule` builder.
func NewRule(factory Factory, request *http.Request, response http.ResponseWriter, rule *model.Rule, template model.Template, actionID string) (Rule, error) {

	const location = "build.NewRule"

	// Create the underlying Common builder
	common, err := NewCommon(factory, request, response, template, actionID)

	if err != nil {
		return Rule{}, derp.Wrap(err, location, "Error creating common builder")
	}

	// Verify that the user is a Domain Owner
	if !common._authorization.DomainOwner {
		return Rule{}, derp.NewForbiddenError(location, "Must be domain owner to continue")
	}

	// Return the Rule builder
	return Rule{
		_rule:  rule,
		Common: common,
	}, nil
}

/******************************************
 * RENDERER INTERFACE
 ******************************************/

// Render generates the string value for this Stream
func (w Rule) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w.action.Steps).Get(w._factory, &w, &buffer)

	if status.Error != nil {
		err := derp.Wrap(status.Error, "build.Rule.Render", "Error generating HTML")
		derp.Report(err)
		return "", err
	}

	// Success!
	status.Apply(w._response)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Rule
func (w Rule) View(actionID string) (template.HTML, error) {

	const location = "build.Rule.View"

	builder, err := NewRule(w._factory, w._request, w._response, w._rule, w._template, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, location, "Error creating Rule builder")
	}

	return builder.Render()
}

func (w Rule) NavigationID() string {
	return "admin"
}

func (w Rule) Permalink() string {
	return w.Hostname() + "/admin/rules/" + w.RuleID()
}

func (w Rule) BasePath() string {
	return "/admin/rules/" + w.RuleID()
}

func (w Rule) Token() string {
	return "rules"
}

func (w Rule) PageTitle() string {
	return "Settings"
}

func (w Rule) object() data.Object {
	return w._rule
}

func (w Rule) objectID() primitive.ObjectID {
	return w._rule.RuleID
}

func (w Rule) objectType() string {
	return "Rule"
}

func (w Rule) schema() schema.Schema {
	return schema.New(model.RuleSchema())
}

func (w Rule) service() service.ModelService {
	return w._factory.Rule()
}

func (w Rule) executeTemplate(writer io.Writer, name string, data any) error {
	return w._template.HTMLTemplate.ExecuteTemplate(writer, name, data)
}

func (w Rule) clone(action string) (Builder, error) {
	return NewRule(w._factory, w._request, w._response, w._rule, w._template, action)
}

/******************************************
 * DATA ACCESSORS
 ******************************************/

func (w Rule) RuleID() string {
	if w._rule == nil {
		return ""
	}
	return w._rule.RuleID.Hex()
}

func (w Rule) Label() string {
	if w._rule == nil {
		return ""
	}
	return w._rule.Label
}

/******************************************
 * QUERY BUILDERS
 ******************************************/

func (w Rule) Rules() *QueryBuilder[model.Rule] {

	query := builder.NewBuilder().
		String("type").
		String("behavior").
		String("trigger")

	criteria := exp.And(
		query.Evaluate(w._request.URL.Query()),
		exp.Equal("userId", w._authorization.UserID),
		exp.Equal("deleteDate", 0),
	)

	result := NewQueryBuilder[model.Rule](w._factory.Rule(), criteria)

	return &result
}

func (w Rule) ServerWideRules() *QueryBuilder[model.Rule] {

	query := builder.NewBuilder().
		String("type").
		String("behavior").
		String("trigger")

	criteria := exp.And(
		query.Evaluate(w._request.URL.Query()),
		exp.Equal("userId", primitive.NilObjectID),
		exp.Equal("deleteDate", 0),
	)

	result := NewQueryBuilder[model.Rule](w._factory.Rule(), criteria)

	return &result
}

func (w Rule) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_admin_rule")
}
