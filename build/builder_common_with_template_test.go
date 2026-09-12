package build

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// TestNewCommonWithTemplate_UnroutedPathIsNotFound asserts that a path this server does not route
// answers 404 rather than 400.
//
// Echo's :param captures every remaining segment when no deeper route matches, so an unrouted path
// reaches this constructor as a multi-segment "action" name.  Production logged 7,096 of these as
// 400 Bad Request, which told well-behaved peers their request was malformed.  BUG-144.
func TestNewCommonWithTemplate_UnroutedPathIsNotFound(t *testing.T) {

	// The guard runs before any dependency is touched, so zero values are enough here.
	unrouted := []string{
		"pub/liked/6a6bf5ebc984688fe7520f78",  // the id of every federated Like
		"pub/shared/68dabe13606bd9d1b263df84", // the id of every federated Announce
		"pub/disliked/68dabe13606bd9d1b263df84",
		"pub/context", // written into Stream.Context by upgrade Version23
		"a/b/c/d/e",   // arbitrary depth
	}

	for _, actionID := range unrouted {
		t.Run(actionID, func(t *testing.T) {
			_, err := NewCommonWithTemplate(nil, nil, nil, nil, model.Template{}, nil, actionID)

			require.Error(t, err)
			require.Equal(t, 404, derp.ErrorCode(err), "an unrouted path must not report as a malformed request")
			require.True(t, derp.IsNotFound(err))
		})
	}
}

// TestNewCommonWithTemplate_UnknownActionIsStillBadRequest pins the case the guard deliberately
// does NOT change: a single-segment action name that the Template does not define.  That one may
// be a broken Template link rather than a visitor typo, so it stays visible in the error log.
func TestNewCommonWithTemplate_UnknownActionIsStillBadRequest(t *testing.T) {

	_, err := NewCommonWithTemplate(nil, nil, nil, nil, model.Template{}, nil, "no-such-action")

	require.Error(t, err)
	require.Equal(t, 400, derp.ErrorCode(err))
}

// TestNewCommonWithTemplate_WithdrawnAndMissingAnswerAlike is the privacy property.
//
// RULE: /pub/liked/<id> is withdrawn on purpose (see TestRoutes_WithdrawnUserCollectionsStayWithdrawn).
// A Response ID is an ObjectID, which embeds a timestamp and is partly guessable, so the answer for a
// Response that EXISTS must be identical to the answer for one that does not -- otherwise the status
// code enumerates a User's private reactions.
func TestNewCommonWithTemplate_WithdrawnAndMissingAnswerAlike(t *testing.T) {

	existing := "pub/liked/6a6bf5ebc984688fe7520f78"
	missing := "pub/liked/000000000000000000000000"

	_, errExisting := NewCommonWithTemplate(nil, nil, nil, nil, model.Template{}, nil, existing)
	_, errMissing := NewCommonWithTemplate(nil, nil, nil, nil, model.Template{}, nil, missing)

	require.Equal(t, derp.ErrorCode(errExisting), derp.ErrorCode(errMissing))
	require.Equal(t, derp.Message(errExisting), derp.Message(errMissing))
}
