package service

/*
func (service *MerchantAccount) paypal_getServerAddress(merchantAccount *model.MerchantAccount) string {

	if merchantAccount.LiveMode {
		return "https://api-m.paypal.com"
	} else {
		return "https://api-m.sandbox.paypal.com"
	}
}

// paypal_processWebhook processes product webhook events from PayPal
func (service *MerchantAccount) paypal_processWebhook(header http.Header, body []byte, merchantAccount *model.MerchantAccount) error {

	const location = "service.MerchantAccount.paypal_processWebhook"
	return derp.NotImplementedError(location)
}

// paypal_refreshMerchantAccount connects/refreshes the PayPal merchant account data
func (service *MerchantAccount) paypal_refreshMerchantAccount(merchantAccount *model.MerchantAccount) error {

	const location = "service.MerchantAccount.paypal_refreshMerchantAccount"

	return derp.BadRequestError(location, "PayPal is not implemented")

		// RULE: Vault MUST have clientId to proceed.  Otherwise,
		// this MMerchantAccount is not fully configured yet.
		if !merchantAccount.Vault.HasString("clientId") {
			return nil
		}

		// RULE: Vault MUST have secretKey to proceed.  Otherwise,
		// this MerchantAccount is not fully configured yet.
		if !merchantAccount.Vault.HasString("secretKey") {
			return nil
		}

		// Decode the encryption key (this should never fail)
		encryptionKey, err := hex.DecodeString(service.encryptionKey)

		if err != nil {
			return derp.Wrap(err, location, "Unable to decode encryption key")
		}

		// Open the Vault to get the clientID and secret key
		vault, err := merchantAccount.Vault.Decrypt(encryptionKey)

		if err != nil {
			return derp.Wrap(err, location, "Error decrypting vault data")
		}

		// Collect variables
		clientID := vault.GetString("clientId")
		secretKey := vault.GetString("secretKey")
		result := make(mapof.Any)

		endpoint := service.paypal_getServerAddress(merchantAccount) + "/v1/oauth2/token"

		// Connect to the PayPal API
		txn := remote.Post(endpoint).
			With(options.BasicAuth(clientID, secretKey)).
			ContentType("application/x-www-form-urlencoded").
			Form("grant_type", "client_credentials").
			Result(&result)

		if err := txn.Send(); err != nil {
			return derp.Wrap(err, location, "Error connecting to PayPal to refresh API key")
		}

		// Update values in the vault
		merchantAccount.Vault.SetString("appId", result.GetString("app_id"))
		merchantAccount.Vault.SetString("accessToken", result.GetString("access_token"))
		merchantAccount.APIKeyExpirationDate = time.Now().Unix() + result.GetInt64("expires_in")

		// Re-encrypt the vault data
		if err := merchantAccount.Vault.Encrypt(encryptionKey); err != nil {
			return derp.Wrap(err, location, "Error encrypting vault data")
		}

		// Success!
		return nil
}

func (service *MerchantAccount) paypal_getProducts(merchantAccount *model.MerchantAccount, productIDs ...string) ([]model.Product, error) {

	const location = "service.MerchantAccount.paypal_getProducts"

	return nil, derp.NotImplementedError(location, "service.MerchantAccount.paypal_getProducts is not implemented")

		// Load the PayPal connection
		connection := model.NewConnection()
		if err := service.connectionService.LoadActiveByType(model.ConnectionProviderPayPal, &connection); err != nil {
			return nil, derp.Wrap(err, location, "Error retrieving PayPal connection")
		}

		// Query PayPal for all Products for this Merchant Account
		endpoint := service.paypal_getServerAddress(merchantAccount) + "/v1/catalogs/products"
		txnResult := mapof.NewAny()
		txn := remote.Get(endpoint).
			//	Query("sort_by", "create_time").
			//	Query("sort_order", "desc").
			ContentType("application/json").
			Header("Prefer", "return=representation").
			With(options.BearerAuth(connection.Token.AccessToken)).
			Result(&txnResult)

		if err := txn.Send(); err != nil {
			return nil, derp.Wrap(err, location, "Error connecting to PayPal API")
		}

		return []model.Product{}, nil

}

func (service *MerchantAccount) paypal_getCheckoutURL(merchantAccount *model.MerchantAccount, remoteProductID string, returnURL string) (string, error) {

	const location = "service.MerchantAccount.paypal_getCheckoutURL"

	return "", derp.NotImplementedError(location, "PayPal is not implemented")

		// Get API Keys from the vault
		apiKeys, err := service.DecryptVault(merchantAccount, "accessToken")

		if err != nil {
			return "", derp.Wrap(err, location, "Error retrieving API keys")
		}

		// Create the checkout URL
		endpoint := service.paypal_getServerAddress(merchantAccount) + "/v1/billing/products/" + remoteProductID
		txnResult := mapof.NewAny()

		txn := remote.Post(endpoint).
			ContentType("application/json").
			With(options.BearerAuth(apiKeys.GetString("accessToken"))).
			Result(&txnResult)

		if err := txn.Send(); err != nil {
			return "", derp.Wrap(err, location, "Error connecting to PayPal API")
		}

		return txnResult.GetString("checkout_url"), nil
}

func (service *MerchantAccount) paypal_getPrivilegeFromCheckoutResponse(queryParams url.Values, merchantAccount *model.MerchantAccount) (model.Privilege, error) {
	return model.Privilege{}, derp.NotImplementedError("service.MerchantAccount.paypal_getIdentityFromCheckoutResponse")
}
*/
