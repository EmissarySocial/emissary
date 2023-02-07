package service

import "github.com/golang-jwt/jwt/v4"

// JWT is a service that generates and validates JWT keys.
type JWT struct {
}

func (service JWT) NewJWTKey() (string, any) {

	// TODO: CRITICAL: Implement a real key generator service here.
	return "k1", []byte("secret")
}

func (service JWT) FindJWTKey(token *jwt.Token) (any, error) {

	// TODO: CRITICAL: Implement a real key lookup here.
	return []byte("secret"), nil
}
