package service

import (
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
	collection data.Collection
	queue      *queue.Queue
}

// NewWebhook returns a new instance of the Webhook service
func NewWebhook() Webhook {
	return Webhook{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

func (service *Webhook) Refresh(collection data.Collection, queue *queue.Queue) {
	service.collection = collection
	service.queue = queue
}

/******************************************
 * Common Methods
 ******************************************/

// New returns a new Webhook that uses the named template.
func (service *Webhook) New() model.Webhook {
	result := model.NewWebhook()
	return result
}

// Count returns the number of records that match the provided criteria
func (service *Webhook) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
}

// Query returns an slice containing all of the Webhooks that match the provided criteria
func (service *Webhook) Query(criteria exp.Expression, options ...option.Option) ([]model.Webhook, error) {
	result := make([]model.Webhook, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the Webhooks that match the provided criteria
func (service *Webhook) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Load retrieves an Webhook from the database
func (service *Webhook) Load(criteria exp.Expression, webhook *model.Webhook) error {

	if err := service.collection.Load(notDeleted(criteria), webhook); err != nil {
		return derp.Wrap(err, "service.Webhook.Load", "Error loading Webhook", criteria)
	}

	return nil
}

// Save adds/updates an Webhook in the database
func (service *Webhook) Save(webhook *model.Webhook, note string) error {

	const location = "service.Webhook.Save"

	// Validate the value (using the global webhook schema) before saving
	if err := service.Schema().Validate(webhook); err != nil {
		return derp.Wrap(err, "service.Webhook.Save", "Error validating Webhook using WebhookSchema", webhook)
	}

	// Try to save the Webhook to the database
	if err := service.collection.Save(webhook, note); err != nil {
		return derp.Wrap(err, location, "Error saving Webhook", webhook, note)
	}

	// Success
	return nil
}

// Delete removes an Webhook from the database (virtual delete)
func (service *Webhook) Delete(webhook *model.Webhook, note string) error {

	// Delete this Webhook
	if err := service.collection.Delete(webhook, note); err != nil {
		return derp.Wrap(err, "service.Webhook.Delete", "Error deleting Webhook", webhook, note)
	}

	// Bueno!!
	return nil
}

// DeleteMany removes all child webhooks from the provided webhook (virtual delete)
func (service *Webhook) DeleteMany(criteria exp.Expression, note string) error {

	it, err := service.List(notDeleted(criteria))

	if err != nil {
		return derp.Wrap(err, "service.Webhook.Delete", "Error listing webhooks to delete", criteria)
	}

	webhook := model.NewWebhook()

	for it.Next(&webhook) {
		if err := service.Delete(&webhook, note); err != nil {
			return derp.Wrap(err, "service.Webhook.Delete", "Error deleting webhook", webhook)
		}
		webhook = model.NewWebhook()
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

func (service *Webhook) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *Webhook) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewWebhook()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Webhook) ObjectSave(object data.Object, note string) error {
	if user, ok := object.(*model.Webhook); ok {
		return service.Save(user, note)
	}
	return derp.InternalError("service.Webhook.ObjectSave", "Invalid object type", object)
}

func (service *Webhook) ObjectDelete(object data.Object, note string) error {
	if user, ok := object.(*model.Webhook); ok {
		return service.Delete(user, note)
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

func (service *Webhook) LoadByID(webhookID primitive.ObjectID, result *model.Webhook) error {
	return service.Load(exp.Equal("_id", webhookID), result)
}

func (service *Webhook) QueryByEvent(event string) ([]model.Webhook, error) {
	return service.Query(exp.Equal("events", event))
}

/******************************************
 * Send Webhooks
 ******************************************/

// Send delivers the webhook to all the external webhook URLs that are listening to the given event
func (service *Webhook) Send(getter model.WebhookDataGetter, events ...string) {

	const location = "service.Webhook.Send"

	go func() {

		for _, event := range events {

			webhooks, err := service.QueryByEvent(event)

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

				// Add the webhook transaction to the Queue, with low priority (32)
				txn := remote.Post(webhook.TargetURL).JSON(data).Queue(service.queue)
				if err := txn.Send(); err != nil {
					derp.Report(derp.Wrap(err, location, "Error sending webhook", webhook, data))
					continue
				}

				log.Trace().Str("event", event).Msg("Webhook sent to " + webhook.TargetURL)
			}
		}
	}()
}
