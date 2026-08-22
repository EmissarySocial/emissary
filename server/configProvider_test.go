package server

import (
	"sync"

	"github.com/EmissarySocial/emissary/config"
)

// stubConfigProvider stands in for the server factory. It is safe to swap the configuration from
// another goroutine, which is what a real configuration reload does.
type stubConfigProvider struct {
	lock  sync.RWMutex
	value config.Config
}

// newStubConfigProvider returns a provider whose configuration names the provided domains.
func newStubConfigProvider(hostnames ...string) *stubConfigProvider {
	stub := &stubConfigProvider{}
	stub.set(hostnames...)
	return stub
}

// Config returns an independent copy of the current configuration, as the real factory does.
func (stub *stubConfigProvider) Config() config.Config {
	stub.lock.RLock()
	defer stub.lock.RUnlock()
	return stub.value.Copy()
}

// set replaces the configured domains, the way a reload from storage would.
func (stub *stubConfigProvider) set(hostnames ...string) {

	value := config.DefaultConfig()

	for index, hostname := range hostnames {
		value.Domains.Put(config.Domain{DomainID: string(rune('a' + index)), Hostname: hostname})
	}

	stub.lock.Lock()
	defer stub.lock.Unlock()
	stub.value = value
}

// setCertificateFolder points the configuration at a certificate directory.
func (stub *stubConfigProvider) setCertificateFolder(folder string) {

	stub.lock.Lock()
	defer stub.lock.Unlock()

	value := stub.value.Copy()
	value.Certificates = map[string]string{"adapter": "FILE", "location": folder}
	stub.value = value
}
