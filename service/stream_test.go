package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestStreamService verifies that the Stream service satisfies the ModelService interface
func TestStreamService(t *testing.T) {
	var service any = &Stream{}
	_, ok := service.(ModelService)
	require.True(t, ok)
}
