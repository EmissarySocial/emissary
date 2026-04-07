package emojikey

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmojiKey(t *testing.T) {

	// Helper function to execute a single test
	do := func(value string, expected [5]Emoji) {

		// Get the EmojiKey
		_, emojiKey := EmojiKey([]byte(value))

		// Compare the generated emoji key to the expected value
		require.Equal(t, expected, emojiKey)
	}

	// Test cases with known inputs and expected outputs
	do("test key A", [5]Emoji{{"🕌", "Mosque"}, {"🍉", "Watermelon"}, {"🧃", "Juice Box"}, {"🩻", "X-Ray"}, {"🥜", "Peanut"}})
	do("test key 2", [5]Emoji{{"🌕", "Full Moon"}, {"🌏", "Globe"}, {"🛶", "Canoe"}, {"🫖", "Teapot"}, {"🦚", "Peacock"}})
}
