package moderation

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// coopReportRequest is the payload sent to Coop's POST /api/v1/report endpoint.
type coopReportRequest struct {
	AccountID string   `json:"accountId"`
	StatusIDs []string `json:"statusIds"`
	Comment   string   `json:"comment"`
	Forward   bool     `json:"forward"`
	Category  string   `json:"category"`
	RuleIDs   []string `json:"ruleIds"`
}

// coopReportResponse is the response returned by Coop's POST /api/v1/report endpoint.
type coopReportResponse struct {
	ID            string `json:"id"`
	ActionTaken   bool   `json:"action_taken"`
	ActionTakenAt string `json:"action_taken_at"`
}

// Coop implements the Moderation interface for a Coop moderation backend.
type Coop struct {
	url               string
	apiKey            string
	webhookPublicKey  string
	userItemTypeID    string
	statusItemTypeID string
}

// NewCoop returns a fully configured Coop moderation backend.
func NewCoop(url, apiKey, webhookPublicKey string) Coop {
	return Coop{
		url:              url,
		apiKey:           apiKey,
		webhookPublicKey: webhookPublicKey,
	}
}

// SubmitReport maps a Mastodon-format report (txn.PostReport) into Coop's schema,
// POSTs it to the configured moderation backend, and maps the response back into
// the Mastodon-format object.Report that the handler expects to return.
func (c *Coop) SubmitReport(auth model.Authorization, report txn.PostReport) (object.Report, error) {

	const location = "moderation.Coop.SubmitReport"

	if c.url == "" {
		return object.Report{}, nil
	}

	if c.apiKey == "" {
		return object.Report{}, derp.BadRequest(location, "Moderation backend URL is configured but no Coop API key is set for this domain")
	}

	// Map Mastodon report → Coop report schema
	coopRequest := coopReportRequest{
		AccountID: report.AccountID,
		StatusIDs: report.StatusIDs,
		Comment:   report.Comment,
		Forward:   report.Forward,
		Category:  report.Category,
		RuleIDs:   report.RuleIDs,
	}

	// POST to Coop
	var coopResp coopReportResponse

	txn := remote.Post(c.url + "/api/v1/report").
		JSON(coopRequest).
		Header("X-API-KEY", c.apiKey).
		Result(&coopResp)

	if err := txn.Send(); err != nil {
		return object.Report{}, derp.Wrap(err, location, "Sending report to moderation backend", c.url)
	}

	// Map Coop response → Mastodon Report object
	return object.Report{
		ID:            coopResp.ID,
		ActionTaken:   coopResp.ActionTaken,
		ActionTakenAt: coopResp.ActionTakenAt,
		Category:      report.Category,
		Comment:       report.Comment,
		Forwarded:     report.Forward,
		CreatedAt:     "", // TODO: set from response or current time
		StatusIDs:     report.StatusIDs,
		RuleIDs:       report.RuleIDs,
	}, nil
}

// VerifySignature verifies a Coop-Signature header against the domain's configured
// webhook public key using RSASSA-PKCS1-v1_5 + SHA-256.
func (c *Coop) VerifySignature(signature string, body []byte) bool {
	// TODO: implement RSASSA-PKCS1-v1_5 + SHA-256 verification using c.webhookPublicKey
	return false
}
