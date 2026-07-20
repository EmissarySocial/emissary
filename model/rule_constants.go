package model

// RuleTypeDomain rules all messages that originate from a specific domain
const RuleTypeDomain = "DOMAIN"

// RuleTypeActor rules all messages from a specific actor
const RuleTypeActor = "ACTOR"

// RuleTypeTag rules all messages carrying a specific hashtag
const RuleTypeTag = "TAG"

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
