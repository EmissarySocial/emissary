package service

import (
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

// JWT is a service that generates and validates JWT keys.
type JWT struct {
}

func (service JWT) NewJWTKey() (string, any) {

	// TODO: CRITICAL: Implement a real key generator service here.
	return "k1", []byte("secret")
}

func (service JWT) FindJWTKey(token *jwt.Token) (any, error) {

	// Key will be stored token.Header "kid" field.
	// TODO: CRITICAL: Implement a real key lookup here.
	return []byte("secret"), nil
}

func (service JWT) Parse(request *http.Request) (*jwt.Token, error) {

	// TODO: CRITICAL: Add WithValidateMthods() to this call.
	return jwt.Parse(request.Header.Get("Authorization"), service.FindJWTKey)
}
