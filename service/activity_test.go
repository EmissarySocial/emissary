package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestActivityService(t *testing.T) {
	var service any = &Activity{}
	_, ok := service.(ModelService)
	require.True(t, ok)
}
