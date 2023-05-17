//go:build local

package sherlock

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
)

func TestLoad_Local(t *testing.T) {

	urlString := "http://localhost/63810bae721f7a33807f25c8"

	meta, err := Load(urlString)

	require.Nil(t, err)
	t.Log(meta)
}

func TestLoad_IndieWeb(t *testing.T) {

	urlString := "https://indieweb.org"

	meta, err := Load(urlString)

	require.Nil(t, err)
	t.Log(meta)
}
