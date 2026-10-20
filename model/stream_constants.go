package model

// Sharing status tokens returned by Stream.SharingStatus
const (
	SharingStatusPublic        = "PUBLIC"
	SharingStatusAuthenticated = "AUTHENTICATED"
	SharingStatusCircles       = "CIRCLES"
	SharingStatusOwners        = "OWNERS"
)

// Reply policies stored in Stream.ReplyPolicy.  An empty policy means "everyone", so every
// Stream saved before the field existed keeps accepting replies.
const (
	ReplyPolicyEveryone = "everyone"
	ReplyPolicyNobody   = "nobody"
)

// JSON-LD properties that publish a Stream's reply policy (REPLY-CONTROLS §1.3).  interactionPolicy
// and its keys are GoToSocial's vocabulary, which FEP-044f adopts.  commentsEnabled is PeerTube's,
// which Lemmy, PieFed, Mbin, Pixelfed, and Mobilizon read.
const (
	PropertyInteractionPolicy = "interactionPolicy"
	PropertyCanReply          = "canReply"
	PropertyAutomaticApproval = "automaticApproval"
	PropertyCommentsEnabled   = "commentsEnabled"
)

// ContextTypeReplyControls declares the reply-policy terms in a Stream's JSON-LD @context, beside
// hannibal's vocab.ContextTypeToot.  Undeclared terms would be dropped by any JSON-LD processor.
var ContextTypeReplyControls = map[string]any{
	"gts":                     "https://gotosocial.org/ns#",
	PropertyInteractionPolicy: map[string]any{"@id": "gts:interactionPolicy", "@type": "@id"},
	PropertyCanReply:          map[string]any{"@id": "gts:canReply", "@type": "@id"},
	PropertyAutomaticApproval: map[string]any{"@id": "gts:automaticApproval", "@type": "@id"},
	"pt":                      "https://joinpeertube.org/ns#",
	PropertyCommentsEnabled:   "pt:commentsEnabled",
}
