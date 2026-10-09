package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

/******************************************
 * Method Parsing
 *
 * Eight Steps take a "method" property, and they guard on it
 * with two different hand-written shapes. An allow-list
 * ("method == post") runs an unknown value for NOTHING; a
 * deny-list ("method != post") runs it for EVERYTHING. Neither
 * reports anything, so a typo used to change when a Step fired
 * with no error to show for it. Validating at parse time is
 * what makes the two shapes interchangeable.
 ******************************************/

func TestParseMethod(t *testing.T) {

	// Every accepted value survives unchanged
	for _, value := range []string{"get", "post", "both"} {
		result, err := parseMethod(mapof.Any{"method": value}, "both")
		require.NoError(t, err)
		require.Equal(t, value, result)
	}

	// Casing is normalized, so "POST" no longer silently misses every comparison
	result, err := parseMethod(mapof.Any{"method": "POST"}, "both")
	require.NoError(t, err)
	require.Equal(t, "post", result)

	// An absent value takes the Step's own default
	result, err = parseMethod(mapof.Any{}, "get")
	require.NoError(t, err)
	require.Equal(t, "get", result)
}

func TestParseMethod_RejectsUnknownValues(t *testing.T) {

	for _, value := range []string{"boht", "put", "delete", "get,post", " get"} {
		_, err := parseMethod(mapof.Any{"method": value}, "both")
		require.Error(t, err, "value: %q", value)
	}
}

// TestParseMethod_RejectedByConstructors confirms the validation reaches Template load
// time through the Steps that read the property, not just the helper.
func TestParseMethod_RejectedByConstructors(t *testing.T) {

	_, err := NewForwardTo(mapof.Any{"url": "/somewhere", "method": "boht"})
	require.Error(t, err)

	_, err = NewRedirectTo(mapof.Any{"url": "/somewhere", "method": "boht"})
	require.Error(t, err)

	_, err = NewViewHTML(mapof.Any{"file": "detail", "method": "boht"})
	require.Error(t, err)

	_, err = NewAddEvent(mapof.Any{"event": "refreshPage", "method": "boht"})
	require.Error(t, err)

	_, err = NewSetHeader(mapof.Any{"name": "X-Test", "value": "1", "method": "boht"})
	require.Error(t, err)
}
