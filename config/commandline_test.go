package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommandLineArgs_Empty(t *testing.T) {

	do := func(location string, expected string) {
		value := CommandLineArgs{Location: location}
		require.Equal(t, expected, value.ConfigDatabase())
	}

	do("mongodb://username:password@localhost:27017/custom", "custom")
	do("mongodb://username:password@some-fancy-url.digitalocean.com:27017", "emissary")

	do("mongodb://localhost:27017/custom", "custom")
	do("mongodb://localhost:27017/emissary", "emissary")
	do("mongodb://localhost:27017/", "emissary")
	do("mongodb://localhost:27017", "emissary")
	do("file://config.json", "")
}
