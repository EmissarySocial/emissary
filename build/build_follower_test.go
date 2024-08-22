package build

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestFollower asserts that builder.Follower implements the Builder interface
func TestFollower(t *testing.T) {
	builder := Builder(Follower{})
	require.NotNil(t, builder)
}
