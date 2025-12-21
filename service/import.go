package service

import (
	"iter"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/random"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	dt "github.com/benpate/domain"
	"github.com/benpate/exp"
	"github.com/benpate/form"
	"github.com/benpate/hannibal/collections"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/sherlock"
	"github.com/benpate/turbine/queue"
	"github.com/davecgh/go-spew/spew"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/oauth2"
)

// Import service helps exports user data to another server
type Import struct {
	activityService   ActivityStream
	attachmentService *Attachment
	importItemService *ImportItem
	locator           ImportableLocator
	queue             *queue.Queue
	host              string
}

// NewImport returns a fully populated Import service
func NewImport(activityService ActivityStream, attachmentService *Attachment, importItemService *ImportItem, locator ImportableLocator, queue *queue.Queue, host string) Import {
	return Import{
		activityService:   activityService,
		attachmentService: attachmentService,
		importItemService: importItemService,
		locator:           locator,
		queue:             queue,
		host:              host,
	}
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Import) collection(session data.Session) data.Collection {
	return session.Collection("Import")
}

func (service *Import) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.Import, error) {
	result := make([]model.Import, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// Count returns the number of records that match the provided criteria
func (service *Import) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// List returns an iterator containing all of the Imports who match the provided criteria
func (service *Import) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the Import records that match the provided criteria
func (service *Import) Range(session data.Session, criteria exp.Expression, options ...option.Option) iter.Seq[model.Import] {

	return func(yield func(model.Import) bool) {

		// Retrieve the Imports from the database
		records, err := service.List(session, criteria, options...)

		// Soft fail.  Report, but do not crash.
		if err != nil {
			derp.Report(derp.Wrap(err, "service.Import.Range", "Unable to create iterator", criteria))
			return
		}

		defer derp.ReportFunc(records.Close)

		// Yield each import to the caller one-by-one
		for record := model.NewImport(); records.Next(&record); record = model.NewImport() {
			if !yield(record) {
				return
			}
		}
	}
}

// Load retrieves an Import from the database
func (service *Import) Load(session data.Session, criteria exp.Expression, record *model.Import) error {

	if err := service.collection(session).Load(notDeleted(criteria), record); err != nil {
		return derp.Wrap(err, "service.Import.Load", "Unable to load Import", criteria)
	}

	return nil
}

// Save adds/updates an Import in the database
func (service *Import) Save(session data.Session, record *model.Import, note string) error {

	const location = "service.Import.Save"

	// Validate the value before saving
	if err := service.Schema().Validate(record); err != nil {
		return derp.Wrap(err, location, "Invalid Import record", record)
	}

	// Execute state changes
	if err := service.calcStateChange(session, record); err != nil {
		return derp.Wrap(err, location, "Unable to calculate state change")
	}

	// Save the import to the database
	if err := service.collection(session).Save(record, note); err != nil {
		return derp.Wrap(err, location, "Unable to save Import", record, note)
	}

	return nil
}

// Delete removes an Import from the database (virtual delete)
func (service *Import) Delete(session data.Session, record *model.Import, note string) error {

	const location = "service.Import.Delete"

	switch record.StateID {

	// If this is an "UNDO", then remove all records associated with this Import
	case model.ImportStateDoUndo:

		if err := service.doUndo(session, record); err != nil {
			return derp.Wrap(err, location, "Unable to undo Import")
		}

	// Otherwise, just remove the import and its items, but not imported records
	default:

		if err := service.importItemService.DeleteByImportID(session, record.UserID, record.ImportID); err != nil {
			return derp.Wrap(err, location, "Unable to delete related records", record.ImportID)
		}
	}

	// Mark this as "DELETED"
	record.StateID = model.ImportStateDeleted

	// Delete this Import
	if err := service.collection(session).Delete(record, note); err != nil {
		return derp.Wrap(err, location, "Unable to delete Import", record, note)
	}

	// Hallelujah
	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Import) ObjectType() string {
	return "Import"
}

// New returns a fully initialized model.Import as a data.Object.
func (service *Import) ObjectNew() data.Object {
	result := model.NewImport()
	return &result
}

// ObjectID returns the ID of a record object
func (service *Import) ObjectID(object data.Object) primitive.ObjectID {

	if record, ok := object.(*model.Import); ok {
		return record.ImportID
	}

	return primitive.NilObjectID
}

func (service *Import) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

func (service *Import) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewImport()
	err := service.Load(session, criteria, &result)
	return &result, err
}

func (service *Import) ObjectSave(session data.Session, object data.Object, note string) error {
	if record, ok := object.(*model.Import); ok {
		return service.Save(session, record, note)
	}
	return derp.InternalError("service.Import.ObjectSave", "Invalid object type", object)
}

func (service *Import) ObjectDelete(session data.Session, object data.Object, note string) error {
	if record, ok := object.(*model.Import); ok {
		return service.Delete(session, record, note)
	}
	return derp.InternalError("service.Import.ObjectDelete", "Invalid object type", object)
}

func (service *Import) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.UnauthorizedError("service.Import.ObjectUserCan", "Not Authorized")
}

func (service *Import) Schema() schema.Schema {
	return schema.New(model.ImportSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

// QueryByUser queries all Import records that match the provided UserID
func (service *Import) QueryByUser(session data.Session, userID primitive.ObjectID) ([]model.Import, error) {
	criteria := exp.Equal("userId", userID)
	return service.Query(session, criteria)
}

// LoadByID loads a single Import record based on the provided UserID and ImportID
func (service *Import) LoadByID(session data.Session, userID primitive.ObjectID, importID primitive.ObjectID, record *model.Import) error {
	criteria := exp.Equal("_id", importID).AndEqual("userId", userID)
	return service.Load(session, criteria, record)
}

// LoadByToken loads a single Import record based on the provided UserID and (string formatted ImportID)
func (service *Import) LoadByToken(session data.Session, userID primitive.ObjectID, token string, record *model.Import) error {

	importID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, "service.Import.LoadByToken", "Import token must be a valid ObjectID", token)
	}

	return service.LoadByID(session, userID, importID, record)
}

// LoadBySourceURL loads a single Import record based on the provided User and SourceURL
func (service *Import) LoadBySourceURL(session data.Session, userID primitive.ObjectID, sourceURL string, record *model.Import) error {
	criteria := exp.Equal("sourceUrl", sourceURL).AndEqual("userId", userID)
	return service.Load(session, criteria, record)
}

/******************************************
 * Custom Actions
 ******************************************/

func (service *Import) SetMessage(session data.Session, record *model.Import, message string) error {
	record.Message = message
	return service.Save(session, record, "Update message")
}

func (service *Import) SetState(session data.Session, record *model.Import, stateID string) error {
	record.StateID = stateID
	record.Message = ""
	return service.Save(session, record, "Update state")
}

/******************************************
 * State Machine
 ******************************************/

// calcStateChange performs additional actions on "transient" states that must be resolved into
// static states before saving
func (service *Import) calcStateChange(session data.Session, record *model.Import) error {

	switch record.StateID {

	case model.ImportStateDoAuthorize:
		return service.doAuthorize(record)

	case model.ImportStateDoImport:
		return service.doImport(record)

	case model.ImportStateDoMove:
		return service.doMove(record)

	case model.ImportStateDoUndo:
		return service.doUndo(session, record)
	}

	return nil
}

// doAuthorize manages the transient state change from "DO-AUTHORIZE"
// to "AUTHORIZING".
func (service *Import) doAuthorize(record *model.Import) error {

	const location = "service.Import.doAuthorize"
	var err error

	// Find the remote actor identified as the Source account
	client := service.activityService.Client()
	actor, err := client.Load(record.SourceID, sherlock.AsActor())

	if err != nil {
		record.StateID = model.ImportStateAuthorizationError
		record.Message = "The account you provided could not be found. Please enter a different account."
		return nil
	}

	// RULE: Require that the remote actor is a "Person"
	if actor.Type() != vocab.ActorTypePerson {
		record.StateID = model.ImportStateAuthorizationError
		record.Message = "The account you provided is not valid because it is a '" + actor.Type() + "' type record. You can only import from 'Person' accounts."
		return nil
	}

	// Generate random OAuth "challenge" data
	record.OAuthChallenge, err = random.GenerateBytes(64)

	if err != nil {
		return derp.Wrap(err, location, "Unable to generate random string")
	}

	// Populate the Import record with the new OAuth configuration data
	record.StateID = model.ImportStateAuthorizing
	record.Message = ""

	record.OAuthConfig = oauth2.Config{
		ClientID:    service.host + "/oauth/metadata", // use new CIMD format (https://cimd.dev)
		RedirectURL: service.OAuthClientCallbackURL(),
		Scopes:      []string{"activitypub_account_portability"},
		Endpoint: oauth2.Endpoint{
			AuthURL:   actor.Endpoints().Get(vocab.EndpointOAuthAuthorization).String(),
			TokenURL:  actor.Endpoints().Get(vocab.EndpointOAuthToken).String(),
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}

	// RULE: AuthURL cannot be empty
	if record.OAuthConfig.Endpoint.AuthURL == "" {
		record.StateID = model.ImportStateAuthorizationError
		record.Message = "The account you provided does not support account migration.  Actors must define an OAuth endpoint to be compatible. (AuthURL missing)"
		return nil
	}

	// RULE: TokenURL cannot be empty
	if record.OAuthConfig.Endpoint.TokenURL == "" {
		record.StateID = model.ImportStateAuthorizationError
		record.Message = "The account you provided does not support account migration.  Actors must define an OAuth endpoint to be compatible. (TokenURL missing)"
		return nil
	}

	// Success
	return nil
}

// doImport manages the transient state change from "DO-IMPORT"
// to "IMPORTING"
func (service *Import) doImport(record *model.Import) error {

	// Start a background task to count all
	service.queue.NewTask(
		"ImportStartup",
		mapof.Any{
			"host":     dt.NameOnly(service.host),
			"userId":   record.UserID,
			"importId": record.ImportID,
		},
	)

	// This message will display in the UX
	record.Message = "Counting Importable Items..."
	record.StateID = model.ImportStateImporting
	return nil
}

// doMove manages the transient state change from "DO-MOVE"
// to "MOVING"
func (service *Import) doMove(record *model.Import) error {

	// Delete OAuth tokens since they're no longer valid
	record.ClearOAuthToken()

	// Update the state to "DONE"
	record.StateID = model.ImportStateDone

	return nil
}

// doUndo manages the transient state change from "DO-UNDO"
// to deleted
func (service *Import) doUndo(session data.Session, record *model.Import) error {

	const location = "service.Import.doUndo"

	// Retrieve all ImportItems to undo
	items, err := service.importItemService.RangeByImportID(session, record.UserID, record.ImportID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to range over ImportItems", record.ImportID)
	}

	// Undo each ImportItem record...
	for item := range items {

		// If this ImportItem was successful, then UNDO the imported record
		if item.StateID == model.ImportItemStateDone {

			if importable, err := service.locator(item.Type); err == nil {

				if err := importable.UndoImport(session, &item); err != nil {
					derp.Report(derp.Wrap(err, location, "Unable to undo imported record"))
				}
			}
		}

		// Delete the ImportItem
		if err := service.importItemService.Delete(session, &item, "Undo"); err != nil {
			derp.Report(derp.Wrap(err, location, "Umable to delete import item", item))
		}
	}

	// Mark this record as "deleted"
	record.DeleteDate = time.Now().Unix()

	// Success!
	return nil
}

/******************************************
 * Import Attachments
 ******************************************/

func (service *Import) ImportAttachments(session data.Session, importRecord *model.Import, importItem *model.ImportItem, object model.AttachmentURLUpdater) error {

	const location = "consumer.importItems_Attachments"

	// Load the /attachments collection using a default (un-cached) client
	client := streams.NewDefaultClient(options.BearerAuth(importRecord.OAuthToken.AccessToken))
	collection, err := client.Load(importItem.ImportURL + "/attachments")

	if err != nil {
		return derp.Wrap(err, location, "Unable to load Attachments")
	}

	spew.Dump(location, collection.Value())

	// Import each attachment in the collection
	for attachment := range collections.RangeDocuments(collection) {

		document := make([]byte, 0)
		txn := remote.Get(attachment.ID()).
			With(options.BearerAuth(importRecord.OAuthToken.AccessToken)).
			With(options.Debug()).
			Result(&document)

		if err := txn.Send(); err != nil {
			return derp.Wrap(err, location, "Unable to retrieve document from source server")
		}

		// Import that attachment
		remoteID, remoteURL, localID, localURL, err := service.attachmentService.Import(
			session,
			importRecord,
			importItem,
			importItem.LocalID,
			document,
		)

		if err != nil {
			return derp.Wrap(err, location, "Unable to import document", remoteID, remoteURL, localID, localURL)
		}

		// Update mappings IF this attachment is named in the containing object
		object.UpdateAttachmentURLs(remoteURL, localURL)
	}

	// Success
	return nil
}

/******************************************
 * Other Calculations
 ******************************************/

// CalcImportPlan locates the best collection to import for each kind of data
// that Emissary supports
func (service *Import) CalcImportPlan(actor streams.Document) sliceof.Object[form.LookupCode] {

	result := sliceof.NewObject[form.LookupCode]()
	migration := actor.Get(vocab.PropertyMigration)

	// First, try to retrieve the main User profile information
	if collection := migration.Get("emissary:user"); collection.NotNil() {
		result.Append(form.LookupCode{
			Group:       "Native",
			Icon:        "patch-check-fill",
			Label:       "User Profile",
			Description: "Maps your old Emissary profile to your new address.",
			Value:       "emissary:user",
			Href:        collection.String(),
		})
	}

	// Retrieve all posts
	if collection := migration.Get("emissary:stream"); collection.NotNil() {
		result.Append(form.LookupCode{
			Group:       "Native",
			Icon:        "patch-check-fill",
			Label:       "Posts",
			Description: "High-fidelity import of all posts and content uploaded to your Emissary profile.",
			Value:       "emissary:stream",
			Href:        collection.String(),
		})

	} else if collection := migration.Get("content"); collection.NotNil() {

		// If we don't have emissary:streams, then try to import standard ActivityPub content
		result.Append(form.LookupCode{
			Group:       "ActivityPub",
			Icon:        "activitypub",
			Label:       "Content",
			Description: "ActivityPub-compatible format. May lose some details in translation",
			Value:       "content",
			Href:        collection.String(),
		})
	}

	// Import Follower records
	if collection := migration.Get("emissary:follower"); collection.NotNil() {
		result.Append(form.LookupCode{
			Group:       "Native",
			Icon:        "patch-check-fill",
			Label:       "Followers",
			Description: "Followers will be notified of this 'Move' but may not choose to re-follow your new account.",
			Value:       "emissary:follower",
			Href:        collection.String(),
		})
	}

	// Import Emissary Inbox Folders
	if collection := migration.Get("emissary:folder"); collection.NotNil() {
		result.Append(form.LookupCode{
			Group:       "Native",
			Icon:        "patch-check-fill",
			Label:       "Inbox Folders",
			Description: "High-fidelity import of all inbox Folders",
			Value:       "emissary:folder",
			Href:        collection.String(),
		})

		// Import Following records
		if collection := migration.Get("emissary:following"); collection.NotNil() {
			result.Append(form.LookupCode{
				Group:       "Native",
				Icon:        "patch-check-fill",
				Label:       "Following",
				Description: "Some accounts may require approval before accepting follow requests from your new account.",
				Value:       "emissary:following",
				Href:        collection.String(),
			})

			// Import Emissary Inbox Messsages IF we have Inbox Folders
			if collection := migration.Get("emissary:inboxMessage"); collection.NotNil() {
				result.Append(form.LookupCode{
					Group:       "Native",
					Icon:        "patch-check-fill",
					Label:       "Inbox Messages",
					Description: "High-fidelity import of your Emissary Inbox",
					Value:       "emissary:inboxMessage",
					Href:        collection.String(),
				})
			}

			// Import Direct Messages IF we have Folders
			if collection := migration.Get("emissary:conversation"); collection.NotNil() {
				result.Append(form.LookupCode{
					Group:       "Native",
					Icon:        "patch-check-fill",
					Label:       "Direct Messages",
					Description: "High-fidelity import of all direct message conversations",
					Value:       "emissary:conversation",
					Href:        collection.String(),
				})
			}
		}

	} else if collection := migration.Get("following"); collection.NotNil() {
		result.Append(form.LookupCode{
			Group:       "ActivityPub",
			Icon:        "activitypub",
			Label:       "Following",
			Description: "Some accounts may require approval before accepting follow requests from your new account.",
			Value:       "following",
			Href:        collection.String(),
		})
	}

	// Try native Outbox Messages
	if collection := migration.Get("emissary:outboxMessage"); collection.NotNil() {
		result.Append(form.LookupCode{
			Group:       "Native",
			Icon:        "patch-check-fill",
			Label:       "Outbox",
			Description: "High-fidelity import of Emissary Outbox.",
			Value:       "emissary:outboxMessage",
			Href:        collection.String(),
		})
	} else if collection := migration.Get("outbox"); collection.NotNil() {
		result.Append(form.LookupCode{
			Group:       "ActivityPub",
			Icon:        "activitypub",
			Label:       "Outbox",
			Description: "ActivityPub-compatible format. May lose some details in translation",
			Value:       "outbox",
			Href:        collection.String(),
		})
	}

	if collection := migration.Get("emissary:rule"); collection.NotNil() {
		result.Append(form.LookupCode{
			Group:       "Native",
			Icon:        "patch-check-fill",
			Label:       "Inbox Rules",
			Description: "High-fidelity import of all custom inbox Rules",
			Value:       "emissary:rule",
			Href:        collection.String(),
		})
	} else if collection := migration.Get("blocked"); collection.NotNil() {
		result.Append(form.LookupCode{
			Group:       "ActivityPub",
			Icon:        "activitypub",
			Label:       "Blocked Collection",
			Description: "Publicly available BLOCKS will be imported",
			Value:       "blocked",
			Href:        collection.String(),
		})
	}

	if collection := migration.Get("emissary:annotation"); collection.NotNil() {
		result.Append(form.LookupCode{
			Group:       "Native",
			Icon:        "patch-check-fill",
			Label:       "Notes",
			Description: "High-fidelity import of all Notes/Annotations",
			Value:       "emissary:annotation",
			Href:        collection.String(),
		})
	}

	// Import Emissay Merchant Accounts
	if collection := migration.Get("emissary:merchantAccount"); collection.NotNil() {
		result.Append(form.LookupCode{
			Group:       "Native",
			Icon:        "patch-check-fill",
			Label:       "Merchant Accounts",
			Description: "High-fidelity import of all Merchant Account settings",
			Value:       "emissary:merchantAccount",
			Href:        collection.String(),
		})

		// Products can be imported IF we have MerchantAccounts
		if collection := migration.Get("emissary:product"); collection.NotNil() {
			result.Append(form.LookupCode{
				Group:       "Native",
				Icon:        "patch-check-fill",
				Label:       "Products",
				Description: "High-fideltiy import of all Products",
				Value:       "emissary:product",
				Href:        collection.String(),
			})

			// Circles can be imported IF we have Products
			if collection := migration.Get("emissary:circle"); collection.NotNil() {
				result.Append(form.LookupCode{
					Group:       "Native",
					Icon:        "patch-check-fill",
					Label:       "Circles",
					Description: "High-fidelity import of all custom Circles",
					Value:       "emissary:circle",
					Href:        collection.String(),
				})

				// Privileges can be imported IF we have Circles
				if collection := migration.Get("emissary:privilege"); collection.NotNil() {
					result.Append(form.LookupCode{
						Group:       "Native",
						Icon:        "patch-check-fill",
						Label:       "Privileges",
						Description: "High-fidelity import of all Privileges/Purchases",
						Value:       "emissary:privilege",
						Href:        collection.String(),
					})
				}
			}
		}
	}

	// Import Emissary Responses
	if collection := migration.Get("emissary:response"); collection.NotNil() {
		result.Append(form.LookupCode{
			Group:       "Native",
			Icon:        "patch-check-fill",
			Label:       "Responses",
			Description: "High-fidelity import of all Responses received",
			Value:       "emissary:response",
			Href:        collection.String(),
		})
	}

	return result
}
