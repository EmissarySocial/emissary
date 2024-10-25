package model

// WebhookEventUserCreate is triggered when a new User record is created
const WebhookEventUserCreate = "user:create"

// WebhookEventUserUpdate is triggered when an existing User record is updated
const WebhookEventUserUpdate = "user:update"

// WebhookEventUserDelete is triggered when an existing User record is deleted
const WebhookEventUserDelete = "user:delete"

// WebhookEventStreamCreate is triggered when a new Stream record is created
const WebhookEventStreamCreate = "stream:create"

// WebhookEventStreamUpdate is triggered when an existing Stream record is updated
const WebhookEventStreamUpdate = "stream:update"

// WebhookEventStreamDelete is triggered when an existing Stream record is deleted
const WebhookEventStreamDelete = "stream:delete"

// WebhookEventStreamPublish is triggered when a Stream is published
const WebhookEventStreamPublish = "stream:publish"

// WebhookEventStreamUnpublish is triggered when a Stream is unpublished
const WebhookEventStreamPublishUndo = "stream:publish:undo"

// WebhookEventStreamSyndicate is triggered when a Stream is syndicated
const WebhookEventStreamSyndicate = "stream:syndicate"

// WebhookEventStreamSyndicateUndo is triggered when a Stream's syndication is undone
const WebhookEventStreamSyndicateUndo = "stream:syndicate:undo"
