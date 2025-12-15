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
	"github.com/benpate/rosetta/schema"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Webhook is a builder for the admin/webhooks page
// It can only be accessed by a Domain Owner
type Webhook struct {
	_webhook *model.Webhook
	CommonWithTemplate
}

// NewWebhook returns a fully initialized `Webhook` builder.
func NewWebhook(factory Factory, session data.Session, request *http.Request, response http.ResponseWriter, template model.Template, webhook *model.Webhook, actionID string) (Webhook, error) {

	const location = "build.NewWebhook"

	// Create the underlying Common builder
	common, err := NewCommonWithTemplate(factory, session, request, response, template, webhook, actionID)

	if err != nil {
		return Webhook{}, derp.Wrap(err, location, "Unable to create common builder")
	}

	// Verify that the webhook is a Domain Owner
	if !common._authorization.DomainOwner {
		return Webhook{}, derp.ForbiddenError(location, "Must be domain owner to continue")
	}

	// Return the Webhook builder
	return Webhook{
		_webhook:           webhook,
		CommonWithTemplate: common,
	}, nil
}

/******************************************
 * Renderer Interface
 ******************************************/

// Render generates the string value for this Webhook
func (w Webhook) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	status := Pipeline(w._action.Steps).Get(w._factory, &w, &buffer)

	if status.Error != nil {
		err := derp.Wrap(status.Error, "build.Webhook.Render", "Unable to generate HTML")
		derp.Report(err)
		return "", err
	}

	// Success!
	status.Apply(w._response)
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Webhook
func (w Webhook) View(actionID string) (template.HTML, error) {

	builder, err := NewWebhook(w._factory, w._session, w._request, w._response, w._template, w._webhook, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, "build.Webhook.View", "Unable to create builder")
	}

	return builder.Render()
}

func (w Webhook) NavigationID() string {
	return "admin"
}

func (w Webhook) Token() string {
	return "webhooks"
}

func (w Webhook) PageTitle() string {
	return "Settings"
}

func (w Webhook) Permalink() string {
	return w.Host() + "/admin/webhooks/" + w.WebhookID()
}

func (w Webhook) BasePath() string {
	return "/admin/webhooks/" + w.WebhookID()
}

func (w Webhook) object() data.Object {
	return w._webhook
}

func (w Webhook) objectID() primitive.ObjectID {
	return w._webhook.WebhookID
}

func (w Webhook) objectType() string {
	return "Webhook"
}

func (w Webhook) schema() schema.Schema {
	return schema.New(model.WebhookSchema())
}

func (w Webhook) service() service.ModelService {
	return w._factory.Webhook()
}

func (w Webhook) clone(action string) (Builder, error) {
	return NewWebhook(w._factory, w._session, w._request, w._response, w._template, w._webhook, action)
}

/******************************************
 * Webhook Data
 ******************************************/

func (w Webhook) WebhookID() string {
	if w._webhook == nil {
		return ""
	}
	return w._webhook.WebhookID.Hex()
}

func (w Webhook) Label() string {
	return w._webhook.Label
}

func (w Webhook) Events() []string {
	return w._webhook.Events
}

func (w Webhook) TargetURL() string {
	return w._webhook.TargetURL
}

/******************************************
 * Other Data Accessors
 ******************************************/

// IsAdminBuilder returns TRUE because Webhook is an admin route.
func (w Webhook) IsAdminBuilder() bool {
	return true
}

/******************************************
 * Query Builders
 ******************************************/

func (w Webhook) Webhooks() *QueryBuilder[model.Webhook] {

	query := builder.NewBuilder().
		String("search", builder.WithAlias("displayName"), builder.WithDefaultOpContains()).
		String("displayName").
		ObjectID("groupId")

	criteria := exp.And(
		query.Evaluate(w._request.URL.Query()),
		exp.Equal("deleteDate", 0),
	)

	result := NewQueryBuilder[model.Webhook](w._factory.Webhook(), w._session, criteria)

	return &result
}

/******************************************
 * Debugging Methods
 ******************************************/

func (w Webhook) debug() {
	log.Debug().Interface("object", w.object()).Msg("builder_admin_webhooks")
}
