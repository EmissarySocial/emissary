package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUser_GetIDFromURL(t *testing.T) {
	require.Equal(t, "benpate", profileAsUserID("http://localhost/@benpate"))
}
