package service

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Webhook service sends outbound webhooks
type Webhook struct {
	factory *Factory
	queue   *queue.Queue
}

// NewWebhook returns a new instance of the Webhook service
func NewWebhook(factory *Factory) Webhook {
	return Webhook{
		factory: factory,
	}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

func (service *Webhook) Refresh(queue *queue.Queue) {
	service.queue = queue
}

/******************************************
 * Common Methods
 ******************************************/

func (service *Webhook) collection(session data.Session) data.Collection {
	return session.Collection("Webhook")
}

// New returns a new Webhook that uses the named template.
func (service *Webhook) New() model.Webhook {
	result := model.NewWebhook()
	return result
}

// Count returns the number of records that match the provided criteria
func (service *Webhook) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns an slice containing all of the Webhooks that match the provided criteria
func (service *Webhook) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.Webhook, error) {
	result := make([]model.Webhook, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the Webhooks that match the provided criteria
func (service *Webhook) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Load retrieves an Webhook from the database
func (service *Webhook) Load(session data.Session, criteria exp.Expression, webhook *model.Webhook) error {

	if err := service.collection(session).Load(notDeleted(criteria), webhook); err != nil {
		return derp.Wrap(err, "service.Webhook.Load", "Unable to load Webhook", criteria)
	}

	return nil
}

// Save adds/updates an Webhook in the database
func (service *Webhook) Save(session data.Session, webhook *model.Webhook, note string) error {

	const location = "service.Webhook.Save"

	// Validate the value (using the global webhook schema) before saving
	if err := service.Schema().Validate(webhook); err != nil {
		return derp.Wrap(err, location, "Unable to validate Webhook using WebhookSchema", webhook)
	}

	// Try to save the Webhook to the database
	if err := service.collection(session).Save(webhook, note); err != nil {
		return derp.Wrap(err, location, "Unable to save Webhook", webhook, note)
	}

	// Success
	return nil
}

// Delete removes an Webhook from the database (virtual delete)
func (service *Webhook) Delete(session data.Session, webhook *model.Webhook, note string) error {

	// Delete this Webhook
	if err := service.collection(session).Delete(webhook, note); err != nil {
		return derp.Wrap(err, "service.Webhook.Delete", "Unable to delete Webhook", webhook, note)
	}

	// Bueno!!
	return nil
}

// DeleteMany removes all child webhooks from the provided webhook (virtual delete)
func (service *Webhook) DeleteMany(session data.Session, criteria exp.Expression, note string) error {

	const location = "service.Webhook.DeleteMany"

	// Get an iterator for every Webhook that matches the criteria
	it, err := service.List(session, notDeleted(criteria))

	if err != nil {
		return derp.Wrap(err, location, "Unable to list webhooks to delete", criteria)
	}

	// Delete every webhook in the Iterator
	for webhook := model.NewWebhook(); it.Next(&webhook); webhook = model.NewWebhook() {

		if err := service.Delete(session, &webhook, note); err != nil {
			return derp.Wrap(err, location, "Unable to delete webhook", webhook)
		}
	}

	return nil
}

/******************************************
 * Generic Data Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Webhook) ObjectType() string {
	return "Webhook"
}

// ObjectNew returns a fully initialized model.Webhook as a data.Object.
func (service *Webhook) ObjectNew() data.Object {
	result := model.NewWebhook()
	return &result
}

func (service *Webhook) ObjectID(object data.Object) primitive.ObjectID {

	if user, ok := object.(*model.Webhook); ok {
		return user.WebhookID
	}

	return primitive.NilObjectID
}

func (service *Webhook) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

func (service *Webhook) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewWebhook()
	err := service.Load(session, criteria, &result)
	return &result, err
}

func (service *Webhook) ObjectSave(session data.Session, object data.Object, note string) error {
	if user, ok := object.(*model.Webhook); ok {
		return service.Save(session, user, note)
	}
	return derp.InternalError("service.Webhook.ObjectSave", "Invalid object type", object)
}

func (service *Webhook) ObjectDelete(session data.Session, object data.Object, note string) error {
	if user, ok := object.(*model.Webhook); ok {
		return service.Delete(session, user, note)
	}
	return derp.InternalError("service.Webhook.ObjectDelete", "Invalid object type", object)
}

func (service *Webhook) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.UnauthorizedError("service.Webhook.ObjectUserCan", "Not Authorized")
}

func (service *Webhook) Schema() schema.Schema {
	return schema.New(model.WebhookSchema())
}

/******************************************
 * Common Queries
 ******************************************/

func (service *Webhook) LoadByID(session data.Session, webhookID primitive.ObjectID, result *model.Webhook) error {
	return service.Load(session, exp.Equal("_id", webhookID), result)
}

func (service *Webhook) QueryByEvent(session data.Session, event string) ([]model.Webhook, error) {
	return service.Query(session, exp.Equal("events", event))
}

/******************************************
 * Send Webhooks
 ******************************************/

// Send delivers the webhook to all the external webhook URLs that are listening to the given event
func (service *Webhook) Send(getter model.WebhookDataGetter, events ...string) {

	const location = "service.Webhook.Send"

	if len(events) == 0 {
		return
	}

	go func() {

		// Create a new (thread-safe) database session
		session, cancel, err := service.factory.Session(time.Minute)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to connect to database"))
			return
		}

		defer cancel()

		for _, event := range events {

			webhooks, err := service.QueryByEvent(session, event)

			if err != nil {
				derp.Report(derp.Wrap(err, location, "Error querying webhooks", event))
				continue
			}

			if len(webhooks) == 0 {
				continue
			}

			// Calculate the data to send
			data := getter.GetWebhookData()
			data["event"] = event

			for _, webhook := range webhooks {

				txn := remote.Post(webhook.TargetURL).JSON(data)

				if err := txn.Send(); err != nil {
					derp.Report(derp.Wrap(err, location, "Unable to send webhook", webhook, data))
					continue
				}

				log.Trace().Str("event", event).Msg("Webhook sent to " + webhook.TargetURL)
			}
		}
	}()
}
