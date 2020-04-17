package model

import "github.com/benpate/data/journal"

// Resource defines any document that is available via this server.
type Resource struct {
	ResourceID string        `json:"resourceId"           bson:"_id"`
	URI        []string      `json:"uri"                  bson:"uri,omitempty"`
	From       string        `json:"from"                 bson:"from,omitempty"`
	Contents   []Content     `json:"content,omitempty"    bson:"content,omitempty"`
	Encrypted  string        `json:"encrypted,omitemtpy"  bson:"encrypted,omitempty"`
	Keys       []ResourceKey `json:"keys,omitemtpy"       bson:"keys,omitemtpy"`

	journal.Journal `json:"journal" bson:"journal"`
}
