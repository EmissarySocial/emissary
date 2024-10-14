package consumer

import "github.com/EmissarySocial/emissary/domain"

type ServerFactory interface {
	ByDomainName(domain string) (*domain.Factory, error)
}
