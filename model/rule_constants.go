package model

// RuleTypeDomain rules all messages that link to a specific domain or URL prefix
const RuleTypeDomain = "DOMAIN"

// RuleTypeUser rules all messages from a specific user
const RuleTypeActor = "ACTOR"

// RuleTypeUser rules all messages that contain a particular phrase (hashtag)
const RuleTypeContent = "CONTENT"

// RuleActionRule rules all contact with a particular user or domain
const RuleActionRule = "BLOCK"

// RuleActionMute prevents inbound messages from a particular user or domain
const RuleActionMute = "MUTE"

// RuleActionLabel allows inbound messages but labels them with a custom message
const RuleActionLabel = "LABEL"
