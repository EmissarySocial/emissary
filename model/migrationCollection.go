package model

// MigrationCollection names one Emissary-native collection in an account export: the key that
// an actor advertises it under, and the last segment of the URL that serves it
type MigrationCollection struct {
	Key  string // Key in the actor's `migration` property, such as "emissary:stream"
	Path string // Last path segment of the export URL, such as "emissary-stream"
}

// MigrationCollections returns every Emissary-native collection that an account export advertises.
// The export routes, the importers, and the import plan must each handle every entry.
func MigrationCollections() []MigrationCollection {

	// The keys are a wire format. Older servers look collections up by key, so never rename one.
	return []MigrationCollection{
		{Key: "emissary:annotation", Path: "emissary-annotation"},
		{Key: "emissary:circle", Path: "emissary-circle"},
		{Key: "emissary:collection", Path: "emissary-collection"},
		{Key: "emissary:collectionItem", Path: "emissary-collection-item"},
		{Key: "emissary:folder", Path: "emissary-folder"},
		{Key: "emissary:follower", Path: "emissary-follower"},
		{Key: "emissary:following", Path: "emissary-following"},
		{Key: "emissary:merchantAccount", Path: "emissary-merchantAccount"},
		{Key: "emissary:newsItem", Path: "emissary-newsFeed"},
		{Key: "emissary:outboxItem", Path: "emissary-outboxItem"},
		{Key: "emissary:outboxMessage", Path: "emissary-outboxMessage"},
		{Key: "emissary:privilege", Path: "emissary-privilege"},
		{Key: "emissary:product", Path: "emissary-product"},
		{Key: "emissary:response", Path: "emissary-response"},
		{Key: "emissary:rule", Path: "emissary-rule"},
		{Key: "emissary:stream", Path: "emissary-stream"},
		{Key: "emissary:user", Path: "emissary-user"},
	}
}
