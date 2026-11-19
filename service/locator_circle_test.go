package service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestLocator_Circle verifies that a circle URL is read back as written, and that nothing else parses as one
func TestLocator_Circle(t *testing.T) {

	locator := Locator{host: "https://example.com"}
	userID := primitive.NewObjectID()
	circleID := primitive.NewObjectID()

	circleURL := locator.CircleURL(userID, circleID)
	require.Equal(t, "https://example.com/@"+userID.Hex()+"/pub/circles/"+circleID.Hex(), circleURL)

	parsedUser, parsedCircle, found := locator.ParseCircle(circleURL)
	require.True(t, found)
	require.Equal(t, userID, parsedUser)
	require.Equal(t, circleID, parsedCircle)

	for _, other := range []string{
		"",
		"https://example.com/@" + userID.Hex(),
		"https://example.com/@" + userID.Hex() + "/pub/followers",
		"https://example.com/@" + userID.Hex() + "/pub/circles/",
		"https://example.com/@" + userID.Hex() + "/pub/circles/not-an-id",
		"https://example.com/@not-an-id/pub/circles/" + circleID.Hex(),
		"https://example.com/@" + userID.Hex() + "/pub/circles/" + circleID.Hex() + "/extra",
		"https://evil.example/@" + userID.Hex() + "/pub/circles/" + circleID.Hex(),
		"https://example.com.evil.example/@" + userID.Hex() + "/pub/circles/" + circleID.Hex(),
		"https://example.com/" + userID.Hex() + "/pub/circles/" + circleID.Hex(),
	} {
		_, _, found := locator.ParseCircle(other)
		require.False(t, found, other)
	}
}
