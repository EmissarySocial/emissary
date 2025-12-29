// go:build localonly

package service

import (
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
	"github.com/benpate/rosetta/mapof"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
)

func TestPayPal_OAuth(t *testing.T) {

	var response mapof.Any
	clientID := "ASJSNmgI1_3dOc5THh5pWfYVMjdwGtEBANUilgF5IjxilefFuLmJIcEZNab80_k63kQdPDjRvbAHpKgv"
	secretKey := "EDlVV8lWh-4li56gtBu-aRALCynj6Brd_Lh8k3WqvZkd38zpDEr2-hvPqdAGNE-nL972AKw9V3ocivxT"

	txn := remote.Post("https://api-m.sandbox.paypal.com/v1/oauth2/token").
		With(options.BasicAuth(clientID, secretKey)).
		ContentType("application/x-www-form-urlencoded").
		Form("grant_type", "client_credentials").
		Result(&response)

	if err := txn.Send(); err != nil {
		derp.Report(err)
		require.Nil(t, err)
	}

	spew.Dump(response)
}

func TestPayPal_API(t *testing.T) {

	clientID := "ASJSNmgI1_3dOc5THh5pWfYVMjdwGtEBANUilgF5IjxilefFuLmJIcEZNab80_k63kQdPDjRvbAHpKgv"
	secretKey := "EDlVV8lWh-4li56gtBu-aRALCynj6Brd_Lh8k3WqvZkd38zpDEr2-hvPqdAGNE-nL972AKw9V3ocivxT"

	spew.Dump(clientID, secretKey)
	// remote.Get("https://sandbox.paypel.com/v1/billing/plans")

}
