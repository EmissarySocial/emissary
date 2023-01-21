package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/channels"
	"github.com/benpate/derp"
)

// Origin service polls external Stream origins for new external Streams.
type Origin struct {
	domainService   *Domain
	streamService   *Stream
	providerService *Provider
}

// NewOrigin returns a fully initialized Origin service
func NewOrigin(domainService *Domain, streamService *Stream, providerService *Provider) Origin {

	result := Origin{
		domainService:   domainService,
		streamService:   streamService,
		providerService: providerService,
	}

	return result
}

/*******************************************
 * Lifecycle Methods
 *******************************************/

func (service *Origin) Refresh(domainService *Domain, streamService *Stream, providerService *Provider) {
	service.domainService = domainService
	service.streamService = streamService
	service.providerService = providerService
}

func (service *Origin) Close() {
}

/*******************************************
 * Polling Methods
 *******************************************/

func (service *Origin) PollAll() <-chan model.Stream {

	clients := service.domainService.ActiveClients()

	clientChannels := make([]<-chan model.Stream, len(clients))

	for index, client := range clients {
		provider, _ := service.providerService.GetProvider(client.ProviderID)
		clientChannels[index] = provider.PollStreams(&client)
	}

	return channels.Merge(clientChannels...)
}

func (service *Origin) Poll(providerID string) <-chan model.Stream {

	client := service.domainService.Client(providerID)
	provider, _ := service.providerService.GetProvider(providerID)

	return provider.PollStreams(&client)
}

func (service *Origin) Save(streams <-chan model.Stream) error {

	for stream := range streams {
		if err := service.streamService.Save(&stream, "Imported from Origin"); err != nil {
			return derp.Wrap(err, "service.Origin.Save", "Error saving stream", stream)
		}
	}

	return nil
}
