package config

import (
	"net/http"
	"time"

	"github.com/benpate/derp"
	"github.com/rs/zerolog/log"
)

// Coop contains configuration for connecting to a Coop moderation instance.
type Coop struct {
	APIKey           string `json:"apiKey"           bson:"apiKey"`           // API key for authenticating outgoing requests to Coop (X-API-KEY header)
	WebhookPublicKey string `json:"webhookPublicKey" bson:"webhookPublicKey"` // PEM-encoded RSA public key used to verify incoming Coop action callbacks (RSASSA-PKCS1-v1_5 + SHA-256)
}

// NewCoop returns a fully initialized (empty) Coop config.
func NewCoop() Coop {
	return Coop{}
}

// IsNil returns TRUE if no Coop backend is configured.
func (c Coop) IsNil() bool {
	return c.APIKey == ""
}

// TestConnection verifies the Coop backend is reachable and accepts the configured API key.
func (c Coop) TestConnection(moderationURL string, timeout time.Duration) error {

	const location = "config.Coop.TestConnection"

	if c.APIKey == "" {
		return derp.Validation("Coop API key is required when Coop is selected as the moderation provider")
	}

	url := moderationURL + "/xrpc/_health"

	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return derp.Wrap(err, location, "Building health check request", url)
	}

	req.Header.Set("X-API-KEY", c.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		return derp.Wrap(err, location, "Unable to reach the Coop moderation backend. Please check the Backend URL in the domain's Moderation settings.", moderationURL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return derp.Validation("Coop backend returned an unexpected status: "+resp.Status, moderationURL, resp.StatusCode)
	}

	log.Info().Str("url", url).Msg("Coop health check passed")

	return nil
}
