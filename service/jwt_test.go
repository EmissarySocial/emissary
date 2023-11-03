package service

import (
	"context"
	"testing"
	"time"

	mockdb "github.com/benpate/data-mock"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestJWT(t *testing.T) {

	// Set up mock server and session
	server := mockdb.New()
	session, err := server.Session(context.TODO())
	require.Nil(t, err)

	collection := session.Collection("test")
	service := NewJWT()
	service.Refresh(collection, []byte("0123456789ABCDEF0123456789ABCDEF"))

	// Create Key1
	name1, value1 := service.NewJWTKey()
	require.Equal(t, time.Now().Format("20060102"), name1)
	require.NotNil(t, value1)

	// Create Key2
	name2, value2 := service.NewJWTKey()
	require.Equal(t, time.Now().Format("20060102"), name2)
	require.NotNil(t, value2)

	// Both values should be the same (because it's still today)
	require.Equal(t, name1, name2)
	require.Equal(t, value1, value2)

	// Let's make a token with our new key
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"ima": "claim",
	})

	token.Header["kid"] = name1

	// Validate that we retrieved the correct key
	value3, err := service.FindJWTKey(token)
	require.Nil(t, err)
	require.Equal(t, value1, value3)
}

func TestJWTResetCache(t *testing.T) {

	// Set up mock server and session
	server := mockdb.New()
	session, err := server.Session(context.TODO())
	require.Nil(t, err)

	collection := session.Collection("test")
	service := NewJWT()
	service.Refresh(collection, []byte("0123456789ABCDEF0123456789ABCDEF"))

	// Create Key1
	name1, value1 := service.NewJWTKey()
	require.Equal(t, time.Now().Format("20060102"), name1)
	require.NotNil(t, value1)

	// Clear everything from the cache
	// so we have to go to the database
	service.cache.Clear()

	// Create Key2
	name2, value2 := service.NewJWTKey()
	require.Equal(t, time.Now().Format("20060102"), name2)
	require.NotNil(t, value2)

	// Both values should be the same (because it's still today)
	require.Equal(t, name1, name2)
	require.Equal(t, value1, value2)

}
