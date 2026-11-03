package mastodon

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSearchWindow confirms client paging values are clamped to Mastodon's documented range
func TestSearchWindow(t *testing.T) {

	tests := []struct {
		name           string
		limit          int64
		offset         int
		expectedLimit  int64
		expectedOffset int
	}{
		{"defaults when unset", 0, 0, searchDefaultLimit, 0},
		{"negative limit uses the default", -5, 0, searchDefaultLimit, 0},
		{"limit within range is kept", 10, 0, 10, 0},
		{"limit above the maximum is capped", 500, 0, searchMaxLimit, 0},
		{"negative offset becomes zero", 10, -3, 10, 0},
		{"offset is kept", 10, 20, 10, 20},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			limit, offset := searchWindow(test.limit, test.offset)
			require.Equal(t, test.expectedLimit, limit)
			require.Equal(t, test.expectedOffset, offset)
		})
	}
}
