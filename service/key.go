package service

import "github.com/golang-jwt/jwt/v4"

type Key struct {
}

func (service Key) NewJWTKey() (string, any) {
	return "k1", []byte("secret")
}

func (service Key) FindJWTKey(token *jwt.Token) (any, error) {
	return []byte("secret"), nil
}
