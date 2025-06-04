package service

import (
	"encoding/hex"
	"iter"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MerchantAccount defines a service that manages all content merchantAccounts created and imported by Users.
type MerchantAccount struct {
	collection        data.Collection
	connectionService *Connection
	jwtService        *JWT
	circleService     *Circle
	identityService   *Identity
	privilegeService  *Privilege
	encryptionKey     string
	host              string
}

// NewMerchantAccount returns a fully initialized MerchantAccount service
func NewMerchantAccount() MerchantAccount {
	return MerchantAccount{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *MerchantAccount) Refresh(collection data.Collection, circleService *Circle, connectionService *Connection, jwtService *JWT, identityService *Identity, privilegeService *Privilege, masterKey string, host string) {
	service.collection = collection
	service.circleService = circleService
	service.connectionService = connectionService
	service.jwtService = jwtService
	service.identityService = identityService
	service.privilegeService = privilegeService
	service.encryptionKey = masterKey
	service.host = host
}

// Close stops any background processes controlled by this service
func (service *MerchantAccount) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *MerchantAccount) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
}

// Query returns an slice of allthe MerchantAccounts that match the provided criteria
func (service *MerchantAccount) Query(criteria exp.Expression, options ...option.Option) ([]model.MerchantAccount, error) {
	result := make([]model.MerchantAccount, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the MerchantAccounts that match the provided criteria
func (service *MerchantAccount) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the MerchantAccount records that match the provided criteria
func (service *MerchantAccount) Range(criteria exp.Expression, options ...option.Option) (iter.Seq[model.MerchantAccount], error) {

	iter, err := service.List(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.MerchantAccount.Range", "Error creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewMerchantAccount), nil
}

// Load retrieves an MerchantAccount from the database
func (service *MerchantAccount) Load(criteria exp.Expression, merchantAccount *model.MerchantAccount) error {

	if err := service.collection.Load(notDeleted(criteria), merchantAccount); err != nil {
		return derp.Wrap(err, "service.MerchantAccount.Load", "Error loading MerchantAccount", criteria)
	}

	return nil
}

// Save adds/updates an MerchantAccount in the database
func (service *MerchantAccount) Save(merchantAccount *model.MerchantAccount, note string) error {

	const location = "service.MerchantAccount.Save"

	// Decode the EncryptionKey
	encryptionKey, err := hex.DecodeString(service.encryptionKey)

	if err != nil {
		return derp.Wrap(err, location, "Error decoding encryption key")
	}

	// Encrypt plaintext values in vault
	if err := merchantAccount.Vault.Encrypt(encryptionKey); err != nil {
		return derp.Wrap(err, location, "Error encrypting vault values")
	}

	// Validate the value before saving
	if err := service.Schema().Validate(merchantAccount); err != nil {
		return derp.Wrap(err, location, "Error validating MerchantAccount")
	}

	// Refresh OAuth connections (if necessary)
	if err := service.RefreshAPIKeys(merchantAccount); err != nil {
		return derp.Wrap(err, location, "Could not connect to "+merchantAccount.Type)
	}

	// Save the merchantAccount to the database
	if err := service.collection.Save(merchantAccount, note); err != nil {
		return derp.Wrap(err, "service.MerchantAccount.Save", "Error saving MerchantAccount", merchantAccount, note)
	}

	return nil
}

// Delete removes an MerchantAccount from the database (virtual delete)
func (service *MerchantAccount) Delete(merchantAccount *model.MerchantAccount, note string) error {

	// Delete this MerchantAccount
	if err := service.collection.Delete(merchantAccount, note); err != nil {
		return derp.Wrap(err, "service.MerchantAccount.Delete", "Error deleting MerchantAccount", merchantAccount, note)
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *MerchantAccount) ObjectType() string {
	return "MerchantAccount"
}

// New returns a fully initialized model.MerchantAccount as a data.Object.
func (service *MerchantAccount) ObjectNew() data.Object {
	result := model.NewMerchantAccount()
	return &result
}

func (service *MerchantAccount) ObjectID(object data.Object) primitive.ObjectID {

	if mention, ok := object.(*model.MerchantAccount); ok {
		return mention.MerchantAccountID
	}

	return primitive.NilObjectID
}

func (service *MerchantAccount) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *MerchantAccount) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewMerchantAccount()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *MerchantAccount) ObjectSave(object data.Object, comment string) error {
	if merchantAccount, ok := object.(*model.MerchantAccount); ok {
		return service.Save(merchantAccount, comment)
	}
	return derp.InternalError("service.MerchantAccount.ObjectSave", "Invalid Object Type", object)
}

func (service *MerchantAccount) ObjectDelete(object data.Object, comment string) error {
	if merchantAccount, ok := object.(*model.MerchantAccount); ok {
		return service.Delete(merchantAccount, comment)
	}
	return derp.InternalError("service.MerchantAccount.ObjectDelete", "Invalid Object Type", object)
}

func (service *MerchantAccount) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.UnauthorizedError("service.MerchantAccount.ObjectUserCan", "Not Authorized")
}

func (service *MerchantAccount) Schema() schema.Schema {
	return schema.New(model.MerchantAccountSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *MerchantAccount) AvailableMerchantAccounts() ([]form.LookupCode, error) {

	const location = "service.MerchantAccount.AvailableMerchantAccounts"

	// Query configured Connections
	connections, err := service.connectionService.QueryByType(model.ConnectionTypeUserPayment)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error loading connections")
	}

	// Map the connections to LookupCodes
	result := slice.Map(connections, func(connection model.Connection) form.LookupCode {
		return connection.LookupCode()
	})

	// Done.
	return result, nil
}

func (service *MerchantAccount) QueryByUser(userID primitive.ObjectID, options ...option.Option) ([]model.MerchantAccount, error) {

	criteria := exp.Equal("userId", userID)

	// Load the Merchant Accounts for this User
	result, err := service.Query(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.MerchantAccount.QueryByUser", "Error loading merchant accounts")
	}

	return result, nil
}

func (service *MerchantAccount) LoadByID(merchantAccountID primitive.ObjectID, merchantAccount *model.MerchantAccount) error {

	criteria := exp.Equal("_id", merchantAccountID)
	return service.Load(criteria, merchantAccount)
}

func (service *MerchantAccount) LoadByUserAndID(userID primitive.ObjectID, merchantAccountID primitive.ObjectID, merchantAccount *model.MerchantAccount) error {

	criteria := exp.Equal("_id", merchantAccountID).AndEqual("userId", userID)
	return service.Load(criteria, merchantAccount)
}

func (service *MerchantAccount) LoadByToken(token string, merchantAccount *model.MerchantAccount) error {

	const location = "service.MerchantAccount.LoadByToken"

	merchantAccountID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, location, "Invalid Token", token)
	}

	return service.LoadByID(merchantAccountID, merchantAccount)
}

func (service *MerchantAccount) LoadByUserAndToken(userID primitive.ObjectID, token string, merchantAccount *model.MerchantAccount) error {

	const location = "service.MerchantAccount.LoadByUserAndToken"

	merchantAccountID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, location, "Invalid Token", token)
	}

	return service.LoadByUserAndID(userID, merchantAccountID, merchantAccount)
}

/******************************************
 * Custom Actions
 ******************************************/

func (service *MerchantAccount) DecryptVault(merchantAccount *model.MerchantAccount, values ...string) (mapof.String, error) {

	const location = "service.MerchantAccount.DecryptVault"

	// Before retrieving the API keys, make sure they are up to date
	if err := service.RefreshAPIKeys(merchantAccount); err != nil {
		return nil, derp.Wrap(err, location, "Error refreshing API keys")
	}

	// Decode the encryption key (this should never fail)
	encryptionKey, err := hex.DecodeString(service.encryptionKey)
	if err != nil {
		return nil, derp.Wrap(err, location, "Error decoding encryption key")
	}

	// Open the Vault to get the clientID and secret key
	vault, err := merchantAccount.Vault.Decrypt(encryptionKey, values...)
	if err != nil {
		return nil, derp.Wrap(err, location, "Error decrypting vault data")
	}

	// Return vault data to the caller
	return vault, nil
}

/******************************************
 * Provider-Specific Methods
 ******************************************/

func (service *MerchantAccount) GetCheckoutURL(merchantAccount *model.MerchantAccount, remoteProductID string, returnURL string) (string, error) {

	switch merchantAccount.Type {

	case model.MerchantAccountTypePayPal:
		return service.paypal_getCheckoutURL(merchantAccount, remoteProductID, returnURL)

	case model.MerchantAccountTypeStripe:
		return service.stripe_getCheckoutURL(merchantAccount, remoteProductID, returnURL)
	}

	return "", derp.BadRequestError("service.MerchantAccount.GetCheckoutURL", "Invalid MerchantAccount Type", merchantAccount.Type)
}

func (service *MerchantAccount) ParseCheckoutResponse(queryParams url.Values, merchantAccount *model.MerchantAccount) (model.Privilege, error) {

	const location = "service.MerchantAccount.ParseCheckoutResponse"

	var getter func(url.Values, *model.MerchantAccount) (model.Privilege, error)

	// Find the appropriate getter function for this MerchantAccount type
	switch merchantAccount.Type {

	case model.MerchantAccountTypePayPal:
		getter = service.paypal_getPrivilegeFromCheckoutResponse

	case model.MerchantAccountTypeStripe:
		getter = service.stripe_getPrivilegeFromCheckoutResponse

	default:
		return model.Privilege{}, derp.BadRequestError(location, "MerchantAccount must be PAYPAL or STRIPE", merchantAccount.Type)
	}

	// Retrieve the Privilege record from the checkout response
	privilege, err := getter(queryParams, merchantAccount)

	if err != nil {
		return model.Privilege{}, derp.Wrap(err, location, "Error processing checkout response")
	}

	// Save the (new?) Privilege record to the database
	if err := service.privilegeService.Save(&privilege, "Created from Checkout"); err != nil {
		return model.Privilege{}, derp.Wrap(err, location, "Error syncing privilege records")
	}

	// Success!
	return privilege, nil
}

func (service *MerchantAccount) ParseCheckoutWebhook(header http.Header, body []byte, merchantAccount *model.MerchantAccount) error {

	const location = "service.MerchantAccount.ParseCheckoutWebhook"

	var parser func(header http.Header, body []byte, merchantAccount *model.MerchantAccount) (model.Privilege, error)

	// Fan out to the appropriate MerchantAccount type
	switch merchantAccount.Type {

	case model.MerchantAccountTypePayPal:
		parser = service.paypal_parseCheckoutWebhook

	case model.MerchantAccountTypeStripe:
		parser = service.stripe_parseCheckoutWebhook

	default:
		return derp.InternalError(location, "Invalid MerchantAccount Type", merchantAccount.Type)
	}

	// Parse the webhook data
	privilege, err := parser(header, body, merchantAccount)

	if err != nil {
		return derp.Wrap(err, location, "Error parsing webhook data")
	}

	// Sync the new/updated privilege record(s)
	if err := service.privilegeService.Save(&privilege, "Updated via WebHook"); err != nil {
		return derp.Wrap(err, location, "Error syncing privilege records")
	}

	// Done.
	return nil
}

func (service *MerchantAccount) RefreshAPIKeys(merchantAccount *model.MerchantAccount) error {

	// RULE: Do not refresh API keys if they will not expire within the next hour
	if merchantAccount.APIKeyExpirationDate > time.Now().Add(-1*time.Hour).Unix() {
		return nil
	}

	switch merchantAccount.Type {

	case model.MerchantAccountTypePayPal:
		return service.paypal_refreshMerchantAccount(merchantAccount)

	case model.MerchantAccountTypeStripe:
		return service.stripe_refreshMerchantAccount(merchantAccount)
	}

	return derp.InternalError("service.MerchantAccount.RefreshMerchantAccount", "Invalid MerchantAccount Type", merchantAccount.Type)

}

// ProductsByUser retrieves all available products configured in the remote MerchantAccount(s) of a specific User
func (service *MerchantAccount) ProductsByUser(userID primitive.ObjectID) (sliceof.Object[model.MerchantAccount], []form.LookupCode, error) {

	// Get all MerchantAccounts for this User
	merchantAccounts, err := service.QueryByUser(userID)

	if err != nil {
		return nil, nil, derp.Wrap(err, "service.MerchantAccount.QueryProductsAsLookupCodes", "Error loading merchant accounts")
	}

	result := make([]form.LookupCode, 0)

	for _, merchantAccount := range merchantAccounts {

		lookupCodes, err := service.getProducts(&merchantAccount)

		if err != nil {
			return nil, nil, derp.Wrap(err, "service.MerchantAccount.QueryProductsAsLookupCodes", "Error loading products for merchant account", merchantAccount)
		}

		result = append(result, lookupCodes...)
	}

	// Sort the final result
	slices.SortFunc(result, form.SortLookupCodeByGroupThenLabel)

	return merchantAccounts, result, nil
}

// ProductsByID retrieves a list of products configured in the remote MerchantAccount(s)
func (service *MerchantAccount) ProductsByID(userID primitive.ObjectID, tokens ...string) ([]form.LookupCode, error) {

	// Collect unique merchantAccount and product IDs
	merchantAccountIDs := cutAndGroup(tokens, ":")
	result := make([]form.LookupCode, 0, len(tokens))

	// Get all Circles
	circles, err := service.circleService.QueryByUser(userID)

	if err != nil {
		return nil, derp.Wrap(err, "service.MerchantAccount.QueryProductsAsLookupCodes", "Error loading circles")
	}

	for _, circle := range circles {
		result = append(result, circle.LookupCode())
	}

	for merchantAccount, productIDs := range merchantAccountIDs {

		// Convert the merchantAccountID to an ObjectID
		merchantAccountID, err := primitive.ObjectIDFromHex(merchantAccount)

		if err != nil {
			return nil, derp.Wrap(err, "service.MerchantAccount.QueryProductsAsLookupCodes", "Invalid MerchantAccount ID", merchantAccountID)
		}

		// Load the MerchantAccount
		merchantAccount := model.NewMerchantAccount()
		if err := service.LoadByUserAndID(userID, merchantAccountID, &merchantAccount); err != nil {
			return nil, derp.Wrap(err, "service.MerchantAccount.QueryProductsAsLookupCodes", "Error loading merchant account", merchantAccountID)
		}

		// Retrieve all of the Products from the remote services
		lookupCodes, err := service.getProducts(&merchantAccount, productIDs...)

		if err != nil {
			return nil, derp.Wrap(err, "service.MerchantAccount.QueryProductsAsLookupCodes", "Error loading products for merchant account", merchantAccount)
		}

		// So far, so good...
		result = append(result, lookupCodes...)
	}

	// Sort the final result
	slices.SortFunc(result, form.SortLookupCodeByGroupThenLabel)

	// Dun dun dunnnnn...
	return result, nil
}

// getProducts lists all of the products configured by a remote service
func (service *MerchantAccount) getProducts(merchantAccount *model.MerchantAccount, productIDs ...string) (sliceof.Object[form.LookupCode], error) {

	switch merchantAccount.Type {

	case model.MerchantAccountTypePayPal:
		return service.paypal_getProducts(merchantAccount, productIDs...)

	case model.MerchantAccountTypeStripe:
		return service.stripe_getPrices(merchantAccount, productIDs...)
	}

	// If we get here, the merchant account type is not supported
	return nil, derp.InternalError("service.MerchantAccount.GetProducts", "Invalid MerchantAccount Type", merchantAccount.Type)
}
