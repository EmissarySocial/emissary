package assanitizer

import (
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/deep"
)

// WithoutBlindRecipients returns a deep copy of an activity with `bto` and `bcc` removed at
// every level of nesting. The original is not changed.
func WithoutBlindRecipients[T any](activity T) (T, error) {

	const location = "assanitizer.WithoutBlindRecipients"

	var zero T

	// RULE: Work on a copy that shares nothing with the original, because delivery still
	// reads the blind recipients from it
	result, err := deep.Copy(activity)

	if err != nil {
		return zero, derp.Wrap(err, location, "Copying activity")
	}

	// RULE: A value that cannot be searched is an error, and nothing is returned, because a
	// blind recipient hidden inside it would be disclosed (AS2 §5.1)
	if err := deep.Remove(result, vocab.PropertyBTo, vocab.PropertyBCC); err != nil {
		return zero, derp.Wrap(err, location, "Removing blind recipients")
	}

	// Blind, but not forgotten
	return result, nil
}
