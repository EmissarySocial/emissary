package db

import "github.com/EmissarySocial/emissary/service"

type Factory interface {
	Model(itemType string) (service.ModelService, error)
}
