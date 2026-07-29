package emojikey

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestParse confirms that summaries split into the correct emojis, including multi-rune sequences
func TestParse(t *testing.T) {

	table := []struct {
		name     string
		summary  string
		expected []Emoji
	}{
		{"empty", "", []Emoji{}},
		{"single", "🐶", []Emoji{{Character: "🐶", Description: "Dog"}}},
		{"standard key", "🕌🍉🧃🩻🥜", []Emoji{
			{Character: "🕌", Description: "Mosque"},
			{Character: "🍉", Description: "Watermelon"},
			{Character: "🧃", Description: "Juice Box"},
			{Character: "🩻", Description: "X-Ray"},
			{Character: "🥜", Description: "Peanut"},
		}},
		{"zwj sequences", "🏳️‍🌈🏴🏴‍☠️", []Emoji{
			{Character: "🏳️‍🌈", Description: "Pride Flag"},
			{Character: "🏴", Description: "Black Flag"},
			{Character: "🏴‍☠️", Description: "Pirate Flag"},
		}},
		{"keycaps", "0️⃣9️⃣", []Emoji{
			{Character: "0️⃣", Description: "Zero"},
			{Character: "9️⃣", Description: "Nine"},
		}},
		{"country flags", "🇯🇵🇧🇷🇨🇦", []Emoji{
			{Character: "🇯🇵", Description: "Japan"},
			{Character: "🇧🇷", Description: "Brazil"},
			{Character: "🇨🇦", Description: "Canada"},
		}},
		{"variation selectors", "☁️❤️", []Emoji{
			{Character: "☁️", Description: "Cloud"},
			{Character: "❤️", Description: "Heart"},
		}},
		{"unknown emoji", "😈", []Emoji{{Character: "😈"}}},
		{"unknown between known", "🐶😈🐱", []Emoji{
			{Character: "🐶", Description: "Dog"},
			{Character: "😈"},
			{Character: "🐱", Description: "Cat"},
		}},
		{"plain text", "ab", []Emoji{{Character: "a"}, {Character: "b"}}},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, Parse(test.summary))
		})
	}
}

// TestParse_FullTable confirms that every canonical character round-trips through Parse
func TestParse_FullTable(t *testing.T) {

	// Join the entire canonical table into one long summary
	emojis := Emojis()
	summary := strings.Builder{}
	for _, emoji := range emojis {
		summary.WriteString(emoji.Character)
	}

	// Parsing it back must reproduce the table exactly
	require.Equal(t, emojis, Parse(summary.String()))
}
