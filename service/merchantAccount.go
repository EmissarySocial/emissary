package service

import (
	"encoding/hex"
	"iter"
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
	connectionService *Connection
	jwtService        *JWT
	circleService     *Circle
	identityService   *Identity
	privilegeService  *Privilege
	productService    *Product
	userService       *User
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
func (service *MerchantAccount) Refresh(circleService *Circle, connectionService *Connection, jwtService *JWT, identityService *Identity, privilegeService *Privilege, productService *Product, userService *User, masterKey string, host string) {
	service.circleService = circleService
	service.connectionService = connectionService
	service.jwtService = jwtService
	service.identityService = identityService
	service.privilegeService = privilegeService
	service.productService = productService
	service.userService = userService
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

func (service *MerchantAccount) collection(session data.Session) data.Collection {
	return session.Collection("MerchantAccount")
}

func (service *MerchantAccount) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns an slice of allthe MerchantAccounts that match the provided criteria
func (service *MerchantAccount) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.MerchantAccount, error) {
	result := make([]model.MerchantAccount, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the MerchantAccounts that match the provided criteria
func (service *MerchantAccount) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the MerchantAccount records that match the provided criteria
func (service *MerchantAccount) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.MerchantAccount], error) {

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.MerchantAccount.Range", "Error creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewMerchantAccount), nil
}

// Load retrieves an MerchantAccount from the database
func (service *MerchantAccount) Load(session data.Session, criteria exp.Expression, merchantAccount *model.MerchantAccount) error {

	if err := service.collection(session).Load(notDeleted(criteria), merchantAccount); err != nil {
		return derp.Wrap(err, "service.MerchantAccount.Load", "Error loading MerchantAccount", criteria)
	}

	return nil
}

// Save adds/updates an MerchantAccount in the database
func (service *MerchantAccount) Save(session data.Session, merchantAccount *model.MerchantAccount, note string) error {

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
	if err := service.Connect(merchantAccount); err != nil {
		return derp.Wrap(err, location, "Could not connect to "+merchantAccount.Type)
	}

	// Save the merchantAccount to the database
	if err := service.collection(session).Save(merchantAccount, note); err != nil {
		return derp.Wrap(err, location, "Error saving MerchantAccount", merchantAccount, note)
	}

	return nil
}

// Delete removes an MerchantAccount from the database (virtual delete)
func (service *MerchantAccount) Delete(session data.Session, merchantAccount *model.MerchantAccount, note string) error {

	// Delete this MerchantAccount
	if err := service.collection(session).Delete(merchantAccount, note); err != nil {
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

func (service *MerchantAccount) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

func (service *MerchantAccount) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewMerchantAccount()
	err := service.Load(session, criteria, &result)
	return &result, err
}

func (service *MerchantAccount) ObjectSave(session data.Session, object data.Object, comment string) error {
	if merchantAccount, ok := object.(*model.MerchantAccount); ok {
		return service.Save(session, merchantAccount, comment)
	}
	return derp.InternalError("service.MerchantAccount.ObjectSave", "Invalid Object Type", object)
}

func (service *MerchantAccount) ObjectDelete(session data.Session, object data.Object, comment string) error {
	if merchantAccount, ok := object.(*model.MerchantAccount); ok {
		return service.Delete(session, merchantAccount, comment)
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

func (service *MerchantAccount) AvailableMerchantAccounts(session data.Session) (sliceof.Object[form.LookupCode], error) {

	const location = "service.MerchantAccount.AvailableMerchantAccounts"

	// Query configured Connections
	connections, err := service.connectionService.QueryActiveByType(session, model.ConnectionTypeUserPayment)

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

func (service *MerchantAccount) QueryByUser(session data.Session, userID primitive.ObjectID, options ...option.Option) (sliceof.Object[model.MerchantAccount], error) {

	const location = "service.MerchantAccount.QueryByUser"

	// RULE: Require a valid UserID
	if userID.IsZero() {
		return nil, derp.ValidationError("UserID cannot be zero")
	}

	criteria := exp.Equal("userId", userID)

	// Load the Merchant Accounts for this User
	result, err := service.Query(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error loading merchant accounts")
	}

	return result, nil
}

func (service *MerchantAccount) LoadByID(session data.Session, merchantAccountID primitive.ObjectID, merchantAccount *model.MerchantAccount) error {

	// RULE: Require a valid MerchantAccountID
	if merchantAccountID.IsZero() {
		return derp.ValidationError("MerchantAccountID cannot be zero")
	}

	criteria := exp.Equal("_id", merchantAccountID)
	return service.Load(session, criteria, merchantAccount)
}

func (service *MerchantAccount) LoadByUserAndID(session data.Session, userID primitive.ObjectID, merchantAccountID primitive.ObjectID, merchantAccount *model.MerchantAccount) error {

	// RULE: Require a valid UserID
	if userID.IsZero() {
		return derp.ValidationError("UserID cannot be zero")
	}

	// RULE: Require a valid MerchantAccountID
	if merchantAccountID.IsZero() {
		return derp.ValidationError("MerchantAccountID cannot be zero")
	}

	criteria := exp.Equal("_id", merchantAccountID).AndEqual("userId", userID)
	return service.Load(session, criteria, merchantAccount)
}

func (service *MerchantAccount) LoadByToken(session data.Session, token string, merchantAccount *model.MerchantAccount) error {

	const location = "service.MerchantAccount.LoadByToken"

	// RULE: Require a valid Token
	if token == "" {
		return derp.ValidationError("Token cannot be empty")
	}

	merchantAccountID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, location, "Invalid Token", token)
	}

	return service.LoadByID(session, merchantAccountID, merchantAccount)
}

func (service *MerchantAccount) LoadByUserAndToken(session data.Session, userID primitive.ObjectID, token string, merchantAccount *model.MerchantAccount) error {

	const location = "service.MerchantAccount.LoadByUserAndToken"

	// RULE: Require a valid UserID
	if userID.IsZero() {
		return derp.ValidationError("UserID cannot be zero")
	}

	// RULE: Require a valid Token
	if token == "" {
		return derp.ValidationError("Token cannot be empty")
	}

	merchantAccountID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, location, "Invalid Token", token)
	}

	return service.LoadByUserAndID(session, userID, merchantAccountID, merchantAccount)
}

/******************************************
 * Custom Actions
 ******************************************/

func (service *MerchantAccount) DecryptVault(merchantAccount *model.MerchantAccount, values ...string) (mapof.String, error) {

	const location = "service.MerchantAccount.DecryptVault"

	/*
		REMOVEING THIS UNTIL WE HAVE AN INTEGRATION THAT DOESN'T USE PERMANENT API KEYS.
		FOR NOW, STRIPE DOESN'T REQUIRE US TO RENEW API KEYS, SO LET'S JUST NOT.
		// Before retrieving the API keys, make sure they are up to date
		if err := service.RefreshAPIKeys(merchantAccount); err != nil {
			return nil, derp.Wrap(err, location, "Error refreshing API keys")
		}
	*/

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

func (service *MerchantAccount) GetCheckoutURL(merchantAccount *model.MerchantAccount, product *model.Product, returnURL string) (string, error) {

	switch merchantAccount.Type {

	// case model.ConnectionProviderPayPal:
	//	return service.paypal_getCheckoutURL(merchantAccount, remoteProductID, returnURL)

	case model.ConnectionProviderStripe:
		return service.stripe_getCheckoutURL(merchantAccount, product, returnURL)

	case model.ConnectionProviderStripeConnect:
		return service.stripe_getCheckoutURL(merchantAccount, product, returnURL)
	}

	return "", derp.BadRequestError("service.MerchantAccount.GetCheckoutURL", "Invalid MerchantAccount Type", merchantAccount.Type)
}

func (service *MerchantAccount) ParseCheckoutResponse(session data.Session, merchantAccount *model.MerchantAccount, product *model.Product, transactionID string, queryParams url.Values) (model.Privilege, error) {

	const location = "service.MerchantAccount.ParseCheckoutResponse"

	var getter func(data.Session, *model.MerchantAccount, *model.Product, string, url.Values) (model.Privilege, error)

	// Find the appropriate getter function for this MerchantAccount type
	switch merchantAccount.Type {

	// case model.ConnectionProviderPayPal:
	//	getter = service.paypal_getPrivilegeFromCheckoutResponse

	case model.ConnectionProviderStripe:
		getter = service.stripe_getPrivilegeFromCheckoutResponse

	case model.ConnectionProviderStripeConnect:
		getter = service.stripe_getPrivilegeFromCheckoutResponse

	default:
		return model.Privilege{}, derp.BadRequestError(location, "MerchantAccount must be PAYPAL or STRIPE", merchantAccount.Type)
	}

	// Retrieve the Privilege record from the checkout response
	privilege, err := getter(session, merchantAccount, product, transactionID, queryParams)

	if err != nil {
		return model.Privilege{}, derp.Wrap(err, location, "Error processing checkout response")
	}

	// Save the (new?) Privilege record to the database
	if err := service.privilegeService.Save(session, &privilege, "Created from Checkout"); err != nil {
		return model.Privilege{}, derp.Wrap(err, location, "Error syncing privilege records")
	}

	// Success!
	return privilege, nil
}

func (service *MerchantAccount) Connect(merchantAccount *model.MerchantAccount) error {

	// RULE: Do not refresh API keys if they will not expire within the next hour
	if merchantAccount.APIKeyExpirationDate > time.Now().Add(-1*time.Hour).Unix() {
		return nil
	}

	switch merchantAccount.Type {

	case model.ConnectionProviderStripe:
		return service.stripe_Connect(merchantAccount)

	case model.ConnectionProviderStripeConnect:
		return service.stripeConnect_Connect(merchantAccount)
	}

	return derp.InternalError("service.MerchantAccount.RefreshMerchantAccount", "Invalid MerchantAccount Type", merchantAccount.Type)

}

// ProductsByUser retrieves all available products configured in the remote MerchantAccount(s) of a specific User
func (service *MerchantAccount) RemoteProductsByUser(session data.Session, userID primitive.ObjectID) (sliceof.Object[model.MerchantAccount], sliceof.Object[model.Product], error) {

	const location = "service.MerchantAccount.RemoteProductsByUser"

	// RULE: Require a valid UserID
	if userID.IsZero() {
		return nil, nil, derp.ValidationError("UserID cannot be zero")
	}

	// Get all MerchantAccounts for this User
	merchantAccounts, err := service.QueryByUser(session, userID)

	if err != nil {
		return nil, nil, derp.Wrap(err, location, "Error loading merchant accounts")
	}

	result := sliceof.NewObject[model.Product]()

	for _, merchantAccount := range merchantAccounts {

		remoteProducts, err := service.getRemoteProducts(&merchantAccount)

		if err != nil {
			return nil, nil, derp.Wrap(err, location, "Error loading products for merchant account", merchantAccount)
		}

		result = append(result, remoteProducts...)
	}

	// Sort the final result
	slices.SortFunc(result, model.SortProducts)

	return merchantAccounts, result, nil
}

// getProducts lists all of the products configured by a remote service
func (service *MerchantAccount) getRemoteProducts(merchantAccount *model.MerchantAccount, productIDs ...string) (sliceof.Object[model.Product], error) {

	const location = "service.MerchantAccount.getRemoteProducts"

	switch merchantAccount.Type {

	// case model.ConnectionProviderPayPal:
	//	return service.paypal_getProducts(merchantAccount, productIDs...)

	case model.ConnectionProviderStripe:
		return service.stripe_getPrices(merchantAccount, productIDs...)

	case model.ConnectionProviderStripeConnect:
		return service.stripe_getPrices(merchantAccount, productIDs...)
	}

	// If we get here, the merchant account type is not supported
	return nil, derp.InternalError(location, "Invalid MerchantAccount Type", merchantAccount.Type)
}

func (service *MerchantAccount) CancelPrivilege(session data.Session, privilege *model.Privilege) error {

	const location = "service.MerchantAccount.CancelPrivilege"

	merchantAccount := model.NewMerchantAccount()

	if err := service.LoadByID(session, privilege.MerchantAccountID, &merchantAccount); err != nil {
		return derp.Wrap(err, "service.MerchantAccount.CancelPrivilege", "Error loading MerchantAccount for Privilege", privilege.PrivilegeID)
	}

	switch merchantAccount.Type {
	case model.ConnectionProviderStripe:
		return service.stripe_CancelPrivilege(&merchantAccount, privilege)

	case model.ConnectionProviderStripeConnect:
		return service.stripe_CancelPrivilege(&merchantAccount, privilege)
	}

	return derp.InternalError(location, "Invalid MerchantAccount Type", merchantAccount.Type)
}
