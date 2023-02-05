package gofed

import (
	"net/url"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestParsePath(t *testing.T) {

	{
		value, _ := url.Parse("http://host/@123456781234567812345678")
		userID, container, activityID, err := ParsePath(value)

		require.Nil(t, err)
		require.Equal(t, knownObjectID("123456781234567812345678"), userID)
		require.Equal(t, model.ActivityStreamContainerUndefined, container)
		require.Equal(t, primitive.NilObjectID, activityID)
	}

	{
		value, _ := url.Parse("http://host/@123456781234567812345678/pub/inbox")
		userID, container, activityID, err := ParsePath(value)

		require.Nil(t, err)
		require.Equal(t, knownObjectID("123456781234567812345678"), userID)
		require.Equal(t, model.ActivityStreamContainerInbox, container)
		require.Equal(t, primitive.NilObjectID, activityID)
	}

	{
		value, _ := url.Parse("http://host/@123456781234567812345678/pub/inbox/876543218765432187654321")
		userID, container, activityID, err := ParsePath(value)

		require.Nil(t, err)
		require.Equal(t, knownObjectID("123456781234567812345678"), userID)
		require.Equal(t, model.ActivityStreamContainerInbox, container)
		require.Equal(t, knownObjectID("876543218765432187654321"), activityID)
	}
}

func TestParsePath_Invalid(t *testing.T) {

	{
		value, _ := url.Parse("http://host/")
		_, _, _, err := ParsePath(value)
		require.NotNil(t, err)
	}

	{
		value, _ := url.Parse("http://host/not-activity-pub")
		_, _, _, err := ParsePath(value)
		require.NotNil(t, err)
	}

	{
		value, _ := url.Parse("http://host/@")
		_, _, _, err := ParsePath(value)
		require.NotNil(t, err)
	}

	{
		value, _ := url.Parse("http://host/@not-an-objectId")
		_, _, _, err := ParsePath(value)
		require.NotNil(t, err)
	}

	{
		value, _ := url.Parse("http://host/@123456781234567812345678/not-pub")
		_, _, _, err := ParsePath(value)
		require.NotNil(t, err)
	}

	{
		value, _ := url.Parse("http://host/@123456781234567812345678/pub/outbox/not-an-objectId")
		_, _, _, err := ParsePath(value)
		require.NotNil(t, err)
	}
}

func TestInboxPath(t *testing.T) {

	{
		value, _ := url.Parse("http://host/@123456781234567812345678/pub/inbox/876543218765432187654321")
		userID, activityID, err := ParseInboxPath(value)

		require.Nil(t, err)
		require.Equal(t, knownObjectID("123456781234567812345678"), userID)
		require.Equal(t, knownObjectID("876543218765432187654321"), activityID)
	}

	{
		value, _ := url.Parse("http://host/@123456781234567812345678/pub/outbox/876543218765432187654321")
		_, _, err := ParseInboxPath(value)

		require.NotNil(t, err)
	}

}

func TestOutPath(t *testing.T) {

	{
		value, _ := url.Parse("http://host/@123456781234567812345678/pub/outbox/876543218765432187654321")
		userID, activityID, err := ParseOutboxPath(value)

		require.Nil(t, err)
		require.Equal(t, knownObjectID("123456781234567812345678"), userID)
		require.Equal(t, knownObjectID("876543218765432187654321"), activityID)
	}

	{
		value, _ := url.Parse("http://host/@123456781234567812345678/pub/inbox/876543218765432187654321")
		_, _, err := ParseOutboxPath(value)

		require.NotNil(t, err)
	}

}

func knownObjectID(value string) primitive.ObjectID {
	result, _ := primitive.ObjectIDFromHex(value)
	return result
}
