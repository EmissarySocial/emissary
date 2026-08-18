package moderation

// Null is a no-op Moderation implementation used when no provider is configured.
type Null struct{}

func (n Null) SubmitReport(report ReportRequest) error {
	return nil
}

func (n Null) VerifySignature(signature string, body []byte) bool {
	return false
}
