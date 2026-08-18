package moderation

import (
	"time"

	"github.com/benpate/derp"
	"github.com/benpate/remote"
)

// coopRelatedItem is a reference to another Coop item, used for RELATED_ITEM fields
// such as author_id. It points to an existing Coop item by its ID and type.
type coopRelatedItem struct {
	ID     string `json:"id"`     // Coop item ID (e.g. the actor URI of the reported user)
	TypeID string `json:"typeId"` // Coop item type ID (e.g. "user")
}

// coopReportRequest is the payload sent to Coop's POST /api/v1/report endpoint.
// It identifies who filed the report, what content is being reported, and why.
// See https://roostorg.github.io/coop/latest/api/report.html
type coopReportRequest struct {
	Reporter     coopReporter     `json:"reporter"`     // who filed the report
	ReportedAt   string           `json:"reportedAt"`   // ISO 8601 timestamp
	ReportedItem coopReportedItem `json:"reportedItem"` // the content being reported
	Comment      string           `json:"comment,omitempty"`
}

// coopReporter identifies the user who filed the report (the reporter, not the
// reported user). In the ActivityPub Flag path, this is Flag.actor — the remote
// server's actor for inbound flags, or the local Emissary user for outbound flags.
type coopReporter struct {
	Kind   string `json:"kind"`   // always "user" (Coop's item kind for accounts)
	TypeID string `json:"typeId"` // Coop user item type ID (configured per domain, e.g. "user")
	ID     string `json:"id"`     // reporter's actor URI or account ID
}

// coopReportedItem identifies the content being reported. In the ActivityPub Flag
// path, this is Flag.object — the URL of the flagged post or actor. The Data map
// carries the content text (if resolvable) and an optional author_id linking back
// to the reported user.
type coopReportedItem struct {
	ID     string         `json:"id"`     // URL/ID of the flagged content
	TypeID string         `json:"typeId"` // Coop status item type ID (configured per domain)
	Data   map[string]any `json:"data"`   // "text": content text, "author_id": coopRelatedItem (optional)
}

// coopReportResponse is the response returned by Coop's POST /api/v1/report endpoint.
type coopReportResponse struct {
	ReportID string `json:"reportId"` // Coop's internal report ID
}

// Coop implements the Moderation interface for a Coop moderation backend.
// It is configured per-domain with the Coop server URL, API key, webhook public
// key (for verifying action callbacks), and the Coop item/action type IDs that
// map Emissary's domain to a Coop organization.
// These types must be defined in the coop dashboard first and then their type IDs can be entered
// in the moderation config for the domain n Emissary.
// See https://roostorg.github.io/coop/latest/api/items.html
type Coop struct {
	url              string // Coop server base URL (e.g. "http://host.docker.internal:9111")
	apiKey           string // Coop API key for authenticating report submissions
	webhookPublicKey string // Coop public key for verifying action webhook callbacks
	userItemTypeID   string // Coop item type ID for user accounts
	statusItemTypeID string // Coop item type ID for status posts
}

// NewCoop returns a fully configured Coop moderation backend.
func NewCoop(url, apiKey, webhookPublicKey, userItemTypeID, statusItemTypeID string) Coop {
	return Coop{
		url:              url,
		apiKey:           apiKey,
		webhookPublicKey: webhookPublicKey,
		userItemTypeID:   userItemTypeID,
		statusItemTypeID: statusItemTypeID,
	}
}

// SubmitReport maps an ActivityPub Flag activity (via ReportRequest) into Coop's
// report schema and POSTs it to the configured moderation backend.
func (c *Coop) SubmitReport(report ReportRequest) error {

	const location = "moderation.Coop.SubmitReport"

	if c.url == "" {
		return nil
	}

	if c.apiKey == "" {
		return derp.BadRequest(location, "Moderation backend URL is configured but no Coop API key is set for this domain")
	}

	reportedItem := coopReportedItem{
		ID:     report.ObjectID,
		TypeID: c.statusItemTypeID,
		Data: map[string]any{
			"text": report.ObjectContent,
		},
	}

	// Only include author_id if we have one
	if report.AuthorID != "" {
		reportedItem.Data["author_id"] = coopRelatedItem{
			ID:     report.AuthorID,
			TypeID: c.userItemTypeID,
		}
	}

	coopRequest := coopReportRequest{
		Reporter: coopReporter{
			Kind:   "user",
			TypeID: c.userItemTypeID,
			ID:     report.ActorID,
		},
		ReportedAt:   time.Now().UTC().Format(time.RFC3339),
		ReportedItem: reportedItem,
		Comment:      report.Comment,
	}

	// POST to Coop
	var coopResp coopReportResponse

	txn := remote.Post(c.url+"/api/v1/report").
		AllowPrivateIPs(true).
		JSON(coopRequest).
		Header("X-API-KEY", c.apiKey).
		Result(&coopResp)

	if err := txn.Send(); err != nil {
		return derp.Wrap(err, location, "Sending report to moderation backend", c.url)
	}

	return nil
}

// VerifySignature verifies a Coop-Signature header against the domain's configured
// webhook public key using RSASSA-PKCS1-v1_5 + SHA-256.
func (c *Coop) VerifySignature(signature string, body []byte) bool {
	// TODO: implement RSASSA-PKCS1-v1_5 + SHA-256 verification using c.webhookPublicKey
	return false
}
