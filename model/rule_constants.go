package model

// RuleTypeDomain rules all messages that link to a specific domain or URL prefix
const RuleTypeDomain = "DOMAIN"

// RuleTypeUser rules all messages from a specific user
const RuleTypeActor = "ACTOR"

// RuleTypeUser rules all messages that contain a particular phrase (hashtag)
const RuleTypeContent = "CONTENT"

// RuleActionBlock rules all contact with a particular user or domain
const RuleActionBlock = "BLOCK"

// RuleActionMute prevents inbound messages from a particular user or domain
const RuleActionMute = "MUTE"

// RuleActionLabel allows inbound messages but labels them with a custom message
const RuleActionLabel = "LABEL"

// RuleOriginAdmin signifies a Rule that was created by a domain administrator
const RuleOriginAdmin = "ADMIN"

// RuleOriginRemote signifies a Rule that was imported from a remote actor
const RuleOriginRemote = "REMOTE"

// RuleOriginUser signifies a Rule that was created by the user
const RuleOriginUser = "USER"

// TagRelationRule identifies a tag that was created by an internal Emissary rule.
const TagRelationRule = "--emissary-rule"
