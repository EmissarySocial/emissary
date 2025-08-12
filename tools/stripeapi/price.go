package stripeapi

import (
	"net/http"
	"strconv"

	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
	"github.com/benpate/rosetta/slice"
	"github.com/stripe/stripe-go/v78"
)

// Prices retrieves all prices from the Stripe API and returns them as a slice
// If no prices are specified, then all active prices are returned.
// https://docs.stripe.com/api/prices/list
func Prices(restrictedKey string, connectedAccountID string, priceIDs ...string) ([]stripe.Price, error) {

	const location = "tools.stripeapi.Products"

	response := stripe.PriceList{}
	result := make([]stripe.Price, 0)

	last := ""
	pageSize := 100

	for {

		// Query the Stripe API for all Prices
		txn := remote.Get("https://api.stripe.com/v1/prices").
			With(options.BearerAuth(restrictedKey)).
			With(ConnectedAccount(connectedAccountID)).
			Query("expand[]", "data.product").
			Query("active", "true").
			Query("limit", strconv.Itoa(pageSize)).
			Result(&response)

		if last != "" {
			txn = txn.Query("starting_after", last)
		}

		if err := txn.Send(); err != nil {
			return nil, derp.Wrap(err, location, "Error connecting to Stripe API", derp.WithCode(http.StatusInternalServerError))
		}

		// NPE check
		if response.Data == nil {
			break
		}

		// If no prices found (this round) then return what we have processed so far
		if len(response.Data) == 0 {
			break
		}

		// Find/Filter prices based on the provided priceIDs
		// If no priceIDs are specified, return all prices
		for _, price := range response.Data {

			// NPE check
			if price == nil {
				continue
			}

			// Filter out inactive prices
			if !price.Active {
				continue
			}

			// NPE check for Product
			if price.Product == nil {
				continue
			}

			// Update the lastID for pagination
			// Doing this BEFORE applying any other app-level filters
			last = price.ID

			// Filter out inactive products
			if !price.Product.Active {
				continue
			}

			// If priceIDs are provided then filter by priceIDs
			if len(priceIDs) > 0 {
				if slice.NotContains(priceIDs, price.ID) {
					continue
				}
			}

			// Result is valid; append to the result.
			result = append(result, *price)
		}

		if len(response.Data) < pageSize {
			// If we received fewer results than the page size, then we are done
			break
		}
	}

	// Done.
	return result, nil
}

// Price loads a Price/Product record from the Stripe API
// https://docs.stripe.com/api/prices/object
func Price(restrictedKey string, connectedAccountID string, priceID string) (stripe.Price, error) {

	const location = "tools.stripeapi.Price"

	// Get the price from the Stripe API
	price := stripe.Price{}
	txn := remote.Get("https://api.stripe.com/v1/prices/"+priceID).
		With(options.BearerAuth(restrictedKey)).
		With(ConnectedAccount(connectedAccountID)).
		Query("expand[]", "product").
		Result(&price)

	if err := txn.Send(); err != nil {
		return stripe.Price{}, derp.Wrap(err, location, "Error connecting to Stripe API", derp.WithCode(http.StatusInternalServerError))
	}

	return price, nil
}
