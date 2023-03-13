package model

// MentionStatusPending represents a Mention that has not yet been validated
const MentionStatusPending = "PENDING"

// MentionStatusValidated represents a Mention that has been validated
const MentionStatusValidated = "VALIDATED"

// MentionStatusInvalid represents a Mention that is not valid
const MentionStatusInvalid = "INVALID"

// MentionTypeStream represents a Mention that references a Stream record
const MentionTypeStream = "Stream"

// MentionTypeUser represents a Mention that references a User record
const MentionTypeUser = "User"
