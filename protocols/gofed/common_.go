package gofed

import "github.com/EmissarySocial/emissary/service"

type Common struct {
	database             Database
	userService          *service.User
	jwtService           *service.JWT
	encryptionKeyService *service.EncryptionKey
	host                 string
}

func NewCommonBehavior(database Database, userService *service.User, encryptionKeyService *service.EncryptionKey, host string) Common {
	return Common{
		database:             database,
		userService:          userService,
		encryptionKeyService: encryptionKeyService,
		host:                 host,
	}
}
