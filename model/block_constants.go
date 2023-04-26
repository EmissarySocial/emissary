package model

// BlockTypeDomain blocks all messages that link to a specific domain or URL prefix
const BlockTypeDomain = "DOMAIN"

// BlockTypeUser blocks all messages from a specific user
const BlockTypeActor = "ACTOR"

// BlockTypeUser blocks all messages that contain a particular phrase (hashtag)
const BlockTypeContent = "CONTENT"

// BlockBehaviorBlock prevents the message from being added to the User's inbox
const BlockBehaviorBlock = "BLOCK"

// BlockBehaviorMute prevents the message from generating a notification and limits its onscreen presence
const BlockBehaviorMute = "MUTE"

// BlockBehaviorAllow allows the message to be added to the User's inbox
const BlockBehaviorAllow = "ALLOW"
