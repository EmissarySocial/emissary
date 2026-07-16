package model

import (
	"slices"

	"github.com/benpate/hannibal/streams"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RuleDispositionNone is the Action of a RuleDisposition that filters nothing.
const RuleDispositionNone = ""

// RuleDisposition is the result of evaluating a document against a set of Rules: the winning Action
// (the most severe of BLOCK/MUTE, or none), the tier and RuleID that produced it (for attribution),
// and every LABEL that matched (regardless of the Action -- labels ride alongside a block, and
// explain it in audit UI).
type RuleDisposition struct {
	Action string             // RuleActionBlock | RuleActionMute | RuleDispositionNone
	Tier   string             // RuleOriginAdmin | RuleOriginUser (of the winning rule; "" if none)
	RuleID primitive.ObjectID // The winning rule, for "why am I seeing this?" UI
	Labels []RuleLabelMatch   // Every LABEL match, from every tier
}

// RuleLabelMatch is one LABEL rule that matched, carried on a RuleDisposition for display and attribution.
type RuleLabelMatch struct {
	RuleID primitive.ObjectID
	Source string // Attribution (FollowingLabel), or "" for the user's own rule
	Label  string // Human-readable label text (Rule.Label)
}

// IsBlocked returns TRUE if the winning Action is BLOCK.
func (disposition RuleDisposition) IsBlocked() bool {
	return disposition.Action == RuleActionBlock
}

// IsMuted returns TRUE if the winning Action is MUTE.
func (disposition RuleDisposition) IsMuted() bool {
	return disposition.Action == RuleActionMute
}

// IsFiltered returns TRUE if the document is blocked OR muted (i.e. it should not create newsfeed
// items). LABEL matches alone do not filter.
func (disposition RuleDisposition) IsFiltered() bool {
	return disposition.IsBlocked() || disposition.IsMuted()
}

// HasLabels returns TRUE if any LABEL rule matched.
func (disposition RuleDisposition) HasLabels() bool {
	return len(disposition.Labels) > 0
}

// NewRuleDisposition is the disposition engine: a PURE function with no I/O and no document
// mutation. It intersects the document's own match keys with the provided Rules and returns the
// resulting RuleDisposition.
//
// `rules` are the candidate summaries the caller pre-fetched (by matchKey, or a whole user's set);
// it re-checks membership, so passing a broader set is safe. `now` is the current Unix time in
// seconds, used to skip expired rules -- it is a parameter (not time.Now) to keep the function pure.
//
// The algorithm, after D8 (no ALLOW) and D10 (two tiers): the winning Action is the maximum severity
// across all matches (BLOCK > MUTE); ties are broken toward the USER tier (D14, attribution only).
// "ADMIN is a floor" is therefore a theorem, not a rule -- a maximum over the union cannot be lowered
// by adding user rules.
func NewRuleDisposition(document streams.Document, rules []RuleSummary, now int64) RuleDisposition {

	keys := DocumentMatchKeys(document)

	result := RuleDisposition{}
	bestSeverity := 0

	for _, rule := range rules {

		// A rule applies only when its key is among the document's keys...
		if (rule.MatchKey == "") || !slices.Contains(keys, rule.MatchKey) {
			continue
		}

		// ...and it has not expired.
		if rule.isExpired(now) {
			continue
		}

		// LABEL rules never filter; they are collected from every tier, even under a block.
		if rule.Action == RuleActionLabel {
			result.Labels = append(result.Labels, RuleLabelMatch{
				RuleID: rule.RuleID,
				Source: rule.FollowingLabel,
				Label:  rule.Label,
			})
			continue
		}

		severity := actionSeverity(rule.Action)

		if severity == 0 {
			continue
		}

		// Higher severity always wins. On a tie, the USER tier wins over ADMIN (D14) -- attribution
		// only, since the Action is identical either way.
		tier := rule.tier()
		beatsWinner := (severity > bestSeverity) ||
			((severity == bestSeverity) && (tier == RuleOriginUser) && (result.Tier == RuleOriginAdmin))

		if beatsWinner {
			bestSeverity = severity
			result.Action = rule.Action
			result.Tier = tier
			result.RuleID = rule.RuleID
		}
	}

	return result
}

// tier returns the attribution tier of this rule: ADMIN for a domain-wide (owner-less) rule, USER
// otherwise. There is no REMOTE tier (D10).
func (rule RuleSummary) tier() string {
	if rule.UserID.IsZero() {
		return RuleOriginAdmin
	}
	return RuleOriginUser
}

// isExpired returns TRUE if this rule has an expiry date that has already passed.
func (rule RuleSummary) isExpired(now int64) bool {
	return (rule.ExpireDate > 0) && (rule.ExpireDate < now)
}

// actionSeverity ranks the filtering actions so the engine can take a maximum. LABEL contributes no
// severity (it never filters); an unrecognized action is inert.
func actionSeverity(action string) int {
	switch action {
	case RuleActionBlock:
		return 3
	case RuleActionMute:
		return 2
	}
	return 0
}
