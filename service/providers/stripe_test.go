package providers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestStripe verifies that the Stripe provider satisfies the ManualProvider interface
func TestStripe(t *testing.T) {

	var stripe any = NewStripe()
	provider := stripe.(ManualProvider)

	require.NotNil(t, provider)
}
