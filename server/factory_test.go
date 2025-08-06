package server

import (
	"testing"

	"github.com/EmissarySocial/emissary/service"
	"github.com/stretchr/testify/require"
)

func TestServerFactory(t *testing.T) {

	var factory service.ServerFactory

	factory = &Factory{}

	require.NotNil(t, factory)

}
