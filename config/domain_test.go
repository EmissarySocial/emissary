package config

import (
	"testing"

	"github.com/benpate/path"
	"github.com/stretchr/testify/require"
)

func TestDomain(t *testing.T) {

	domain := Domain{
		Label:         "Test",
		Hostname:      "test.com",
		ConnectString: "127.0.0.1",
	}

	require.Equal(t, "Test", path.Get(domain, "label"))
	require.Equal(t, "test.com", path.Get(domain, "hostname"))
	require.Equal(t, "127.0.0.1", path.Get(domain, "connectString"))
}
