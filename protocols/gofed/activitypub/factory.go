package activitypub

import "github.com/EmissarySocial/emissary/service"

type Factory interface {
	Model(string) (service.ModelService, error)
}
