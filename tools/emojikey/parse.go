package emojikey

import (
	"sync"
	"unicode/utf8"
)

// tableIndex returns the character → description index and the rune count of the
// longest character in the canonical table, built once on first use.
var tableIndex = sync.OnceValues(func() (map[string]string, int) {

	// Index every canonical character, tracking the longest one
	emojis := Emojis()
	index := make(map[string]string, len(emojis))
	maxRunes := 0

	for _, emoji := range emojis {
		index[emoji.Character] = emoji.Description

		if runeCount := utf8.RuneCountInString(emoji.Character); runeCount > maxRunes {
			maxRunes = runeCount
		}
	}

	// Yer a wizard, Harry.
	return index, maxRunes
})

// Parse splits a client-computed EmojiKey string into the emojis that make it up.
func Parse(summary string) []Emoji {

	index, maxRunes := tableIndex()
	result := make([]Emoji, 0, 8)

	for position := 0; position < len(summary); {

		// Find the longest canonical character that begins at this position.
		// Longest-first matters: the table holds multi-rune sequences whose
		// prefixes are also in the table (🏴 vs 🏴‍☠️), so the first match is
		// not always the right one.
		matchEnd := 0
		matchDescription := ""
		end := position

		for runeCount := 0; runeCount < maxRunes && end < len(summary); runeCount++ {
			_, size := utf8.DecodeRuneInString(summary[end:])
			end += size

			if description, found := index[summary[position:end]]; found {
				matchEnd = end
				matchDescription = description
			}
		}

		// Emit the matched emoji and continue after it
		if matchEnd > position {
			result = append(result, Emoji{Character: summary[position:matchEnd], Description: matchDescription})
			position = matchEnd
			continue
		}

		// Unknown characters pass through one rune at a time, with no description.
		// This keeps summaries from newer client tables displayable instead of breaking.
		_, size := utf8.DecodeRuneInString(summary[position:])
		result = append(result, Emoji{Character: summary[position : position+size]})
		position += size
	}

	// Great Scott!
	return result
}
