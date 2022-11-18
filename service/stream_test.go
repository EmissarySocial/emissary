package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStreamService(t *testing.T) {
	var service any = &Stream{}
	_, ok := service.(ModelService)
	require.True(t, ok)
}
