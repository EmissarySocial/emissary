package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToToken(t *testing.T) {

	do := func(input, expected string) {
		require.Equal(t, ToToken(input), expected)
	}

	do("Hello, World!", "hello-world")
	do("Hägen Däs", "hägen-däs")
	do("Æthelflad", "æthelflad")
}
