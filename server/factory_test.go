package server

import (
	"testing"

	"github.com/EmissarySocial/emissary/service"
	"github.com/stretchr/testify/require"
)

func TestServerFactory(t *testing.T) {
	var factory service.ServerFactory = &Factory{}
	require.NotNil(t, factory)
}
