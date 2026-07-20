package model

import (
	"slices"

	"github.com/benpate/hannibal/metadata"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RuleDispositionNone is the Action of a RuleDisposition that filters nothing.
const RuleDispositionNone = ""

// RuleDisposition is the result of evaluating a document against a set of Rules: the winning Action
// (the most severe of BLOCK/MUTE, or none), the tier and RuleID that produced it (for attribution),
// and every LABEL that matched (regardless of the Action -- labels ride alongside a block, and
// explain it in audit UI). The bson tags let an InboxActivity persist its receive-time disposition.
//
// BOUNDARY: RuleDisposition is the DECISION -- machine fields, severity ordering, persistable --
// consumed by everything the server must DO (refuse, suppress, halt delivery, purge). Its display
// twin is metadata.LabelSet -- lossy and never persisted -- produced ONLY by LabelSet() below.
type RuleDisposition struct {
	Action string             `bson:"action,omitempty"` // RuleActionBlock | RuleActionMute | RuleDispositionNone
	Tier   string             `bson:"tier,omitempty"`   // RuleOriginAdmin | RuleOriginUser (of the winning rule; "" if none)
	RuleID primitive.ObjectID `bson:"ruleId,omitempty"` // The winning rule, for "why am I seeing this?" UI
	Labels []RuleLabelMatch   `bson:"labels,omitempty"` // Every LABEL match, from every tier
}

// RuleLabelMatch is one LABEL rule that matched, carried on a RuleDisposition for display and attribution.
type RuleLabelMatch struct {
	RuleID primitive.ObjectID `bson:"ruleId,omitempty"` // The LABEL rule that matched
	Tier   string             `bson:"tier,omitempty"`   // RuleOriginAdmin | RuleOriginUser (rows persisted before this field read back as "", which links nowhere)
	Source string             `bson:"source,omitempty"` // Attribution (FollowingLabel), or "" for the user's own rule
	Label  string             `bson:"label,omitempty"`  // Human-readable label text (Rule.Label)
}

// IsZero returns TRUE if this disposition filters nothing and carries no labels. The mongo driver
// uses it (as a bsoncodec.Zeroer) to honor `omitempty` on struct fields, so a clean disposition
// writes no field at all -- and rows stored before the field existed read back as clean.
func (disposition RuleDisposition) IsZero() bool {
	return disposition.Action == "" &&
		disposition.Tier == "" &&
		disposition.RuleID.IsZero() &&
		len(disposition.Labels) == 0
}

// Merge combines this disposition with another: the more severe Action wins (ties keep the
// RECEIVER's attribution -- call as current.Merge(persisted) so a live rule outranks a possibly
// deleted one), and LABEL matches are unioned, deduplicated by RuleID. Neither input is modified.
func (disposition RuleDisposition) Merge(other RuleDisposition) RuleDisposition {

	result := disposition
	result.Labels = slices.Clone(disposition.Labels)

	// The more severe Action wins; on a tie, the receiver's attribution stands.
	if actionSeverity(other.Action) > actionSeverity(disposition.Action) {
		result.Action = other.Action
		result.Tier = other.Tier
		result.RuleID = other.RuleID
	}

	// Union the LABEL matches, deduplicated by RuleID
	for _, label := range other.Labels {
		if !result.hasLabelRule(label.RuleID) {
			result.Labels = append(result.Labels, label)
		}
	}

	return result
}

// hasLabelRule returns TRUE if this disposition already carries a LABEL match from the given Rule.
func (disposition RuleDisposition) hasLabelRule(ruleID primitive.ObjectID) bool {
	return slices.ContainsFunc(disposition.Labels, func(label RuleLabelMatch) bool {
		return label.RuleID == ruleID
	})
}

// ApplyLabels writes this disposition's LabelSet into the target JSON-LD map under the reserved
// PropertyEmissaryLabels key. Any existing value is ALWAYS deleted first -- the anti-spoofing
// backstop for rows stored before ingress stripping existed -- and the key is set only when there
// is something server-generated to say.
func (disposition RuleDisposition) ApplyLabels(target mapof.Any) {

	// RULE: the reserved key is server-generated only; whatever is there now is not ours.
	delete(target, PropertyEmissaryLabels)

	// A clean disposition has nothing to add
	if disposition.IsZero() {
		return
	}

	// Serialize the LabelSet as [{value, href, isHidden}]
	labelSet := disposition.LabelSet()
	labels := make([]mapof.Any, 0, len(labelSet))

	for _, label := range labelSet {

		entry := mapof.Any{
			"value":    label.Value,
			"isHidden": label.IsHidden,
		}

		if label.Href != "" {
			entry["href"] = label.Href
		}

		labels = append(labels, entry)
	}

	// Nothing to say after all? Then say nothing.
	if len(labels) == 0 {
		return
	}

	target[PropertyEmissaryLabels] = labels
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

// LabelSet renders this disposition as the per-viewer metadata.LabelSet that rides on a document's
// Metadata: a leading hidden Label when the winning Action filters (block or mute), then one
// annotation per matched LABEL rule. Hidden-first matches the set's display convention.
//
// This one-way derivation is the ONLY producer of a LabelSet anywhere: the decision object stays
// authoritative, and the display set can never say more than the decision did.
func (disposition RuleDisposition) LabelSet() metadata.LabelSet {

	result := make(metadata.LabelSet, 0, len(disposition.Labels)+1)

	// A filtering Action (block or mute) hides the document; its Value names the action and tier.
	if disposition.IsFiltered() {
		result = append(result, metadata.Label{
			Value:    disposition.hiddenReason(),
			Href:     ruleEditHref(disposition.Tier, disposition.RuleID),
			IsHidden: true,
		})
	}

	// Every LABEL match annotates without hiding.
	for _, label := range disposition.Labels {
		if label.Label != "" {
			result = append(result, metadata.Label{
				Value: label.Label,
				Href:  ruleEditHref(label.Tier, label.RuleID),
			})
		}
	}

	return result
}

// ruleEditHref returns the attribution link for a matched rule: USER-tier rules link to the
// viewer's own rule editor ("delete your own rule"), ADMIN-tier rules link nowhere -- they are
// attributed in text ("by server policy") but are not user-removable (D8/D9). Deleting the linked
// rule does NOT promise a reveal: another rule may still cover the item, and the next render
// re-evaluates from scratch (D14).
func ruleEditHref(tier string, ruleID primitive.ObjectID) string {

	if tier == RuleOriginUser && !ruleID.IsZero() {
		return "/@me/settings/rule-edit?ruleId=" + ruleID.Hex()
	}

	return ""
}

// hiddenReason composes the display text for a filtering disposition, naming both what happened
// (blocked or muted) and who is responsible (server policy, or the viewer's own rules).
func (disposition RuleDisposition) hiddenReason() string {

	action := "Filtered"

	switch disposition.Action {
	case RuleActionBlock:
		action = "Blocked"
	case RuleActionMute:
		action = "Muted"
	}

	switch disposition.Tier {
	case RuleOriginAdmin:
		return action + " by server policy"
	case RuleOriginUser:
		return action + " by your rules"
	}

	return action
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
	return NewRuleDispositionForKeys(DocumentMatchKeys(document), rules, now)
}

// NewRuleDispositionForKeys is the disposition engine over a pre-computed key set. Callers that hold
// a whole document should use NewRuleDisposition; the wire gate uses this directly because it knows
// only the actor/domain keys (a claimed actor plus the signature keyId host), never the full document.
// The algorithm is identical -- intersect `keys` with `rules`, take the max severity, ties to USER.
func NewRuleDispositionForKeys(keys []string, rules []RuleSummary, now int64) RuleDisposition {

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
				Tier:   rule.tier(),
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
