package service

import (
	"github.com/benpate/derp"
)

// Export service helps exports user data to another server
type Export struct {
	factory *Factory
}

// NewExport returns a fully populated Export service
func NewExport(factory *Factory) Export {
	return Export{
		factory: factory,
	}
}

func (service *Export) FindService(collectionName string) (Exportable, error) {

	const location = "service.Export.FindService"

	switch collectionName {
	case "outbox":
	case "content":
	case "following":
	case "blocked":
	case "emissary-annotation":
	case "emissary-attachment":
	case "emissary-circle":
	case "emissary-conversation":
	case "emissary-folder":
	case "emissary-follower":
	case "emissary-following":
	case "emissary-mention":
	case "emissary-merchantAccount":
	case "emissary-message":
	case "emissary-outboxMessage":
	case "emissary-privilege":
	case "emissary-product":
	case "emissary-response":
	case "emissary-rule":
	case "emissary-stream":
		return service.factory.Stream(), nil

	case "emissary-streamWidget":
	}

	return nil, derp.NotFound(location, "Collection not found", collectionName)
}

/*
Model Objects tied to Users:
----------------------------

Annotation
Circle
Conversation (still building)
Folder
Follower (via ParentID)
Following
MerchantAccount
Message
OutboxMessage
Privilege
-> Identity (may require tricky merge)
Product
Response (still used?)
Rule

Stream (via ParentIDs)
-> Attachment
-> Mention
-> StreamWidget

User (obv.)

*/
