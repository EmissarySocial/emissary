package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
)

// GetCheckout initiates a checkout session with the provided MerchantAccount and Product.
func GetCheckout(ctx *steranko.Context, factory *service.Factory, session data.Session, merchantAccount *model.MerchantAccount, product *model.Product) error {

	const location = "handler.GetCheckout"

	// RULE: If the buyer is already signed in as a guest Identity, then lock the checkout email to
	// that Identity's verified address. This keeps a signed-in guest from purchasing under (and being
	// redirected toward) a different email. Anonymous buyers pass an empty value and enter their own
	// email at Stripe.
	customerEmail := ""

	if authorization := getAuthorization(ctx); !authorization.IdentityID.IsZero() {
		identity := model.NewIdentity()
		if err := factory.Identity().LoadByID(session, authorization.IdentityID, &identity); err == nil {
			customerEmail = identity.EmailAddress
		}
	}

	// Create a "checkout" session, and generate a URL where the user will checkout
	returnURL := ctx.QueryParam("return")
	merchantAccountService := factory.MerchantAccount()
	checkoutURL, err := merchantAccountService.GetCheckoutURL(merchantAccount, product, returnURL, customerEmail)

	if err != nil {
		return derp.Wrap(err, location, "Retrieving checkout URL")
	}

	// Forward the client to the checkout URL
	return ctx.Redirect(http.StatusTemporaryRedirect, checkoutURL)
}

// GetCheckoutResponse collects the confirmation data from a successful checkout, attaches the
// purchased Privilege to the buyer's email Identity, and grants a guest session ONLY when the buyer
// has already proven ownership of that email.
func GetCheckoutResponse(ctx *steranko.Context, factory *service.Factory, session data.Session, merchantAccount *model.MerchantAccount, product *model.Product) error {

	const location = "handler.GetCheckoutResponse"

	// Verify the Checkout Session and attach the Privilege to the email Identity
	merchantAccountService := factory.MerchantAccount()
	privilege, err := merchantAccountService.ParseCheckoutResponse(session, merchantAccount, product, ctx.QueryParam("transactionId"), ctx.QueryParams())

	if err != nil {
		return derp.Wrap(err, location, "Retrieving checkout URL")
	}

	// RULE: If the buyer is already signed in as this Identity, then they have already proven that
	// they own its email. Send them to their profile to see the newly purchased Privilege.
	authorization := getAuthorization(ctx)

	if authorization.IdentityID == privilege.IdentityID {
		return ctx.Redirect(http.StatusSeeOther, "/@guest")
	}

	// RULE: Otherwise, the buyer has NOT proven that they own the email that Stripe reported. Stripe
	// does not verify customer emails, so authenticating the buyer as a pre-existing Identity from
	// that email would be an account takeover. Require the standard guest-code (OTP) flow instead:
	// send a one-time signin link to the email on the Privilege, which the buyer must click to claim
	// the purchase and sign in. Do NOT set a session cookie here.
	identityService := factory.Identity()

	if err := identityService.SendGuestCode(session, nil, privilege.IdentifierType, privilege.IdentifierValue); err != nil {
		return derp.Wrap(err, location, "Sending guest code to claim purchase")
	}

	// Tell the buyer to check their inbox to claim the purchase.
	return executeDomainTemplate(ctx, factory, session, "checkout-claim")
}
