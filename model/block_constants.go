package model

// BlockTypeDomain blocks all messages that link to a specific domain or URL prefix
const BlockTypeDomain = "DOMAIN"

// BlockTypeUser blocks all messages from a specific user
const BlockTypeActor = "ACTOR"

// BlockTypeUser blocks all messages that contain a particular phrase (hashtag)
const BlockTypeContent = "CONTENT"

// BlockActionBlock blocks all contact with a particular user or domain
const BlockActionBlock = "BLOCK"

// BlockActionMute prevents inbound messages from a particular user or domain
const BlockActionMute = "MUTE"
