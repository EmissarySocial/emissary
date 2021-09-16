package steranko

import "github.com/golang-jwt/jwt/v4"

type KeyService interface {
	NewJWTKey() (string, interface{})
	FindJWTKey(*jwt.Token) (interface{}, error)
}
