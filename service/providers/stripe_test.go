package providers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStripe(t *testing.T) {

	var stripe any = NewStripe()
	provider := stripe.(ManualProvider)

	require.NotNil(t, provider)
}
