package paypal

// APIHost returns the PayPal API endpoint for either live or sandbox mode
func APIHost(liveMode bool) string {
	if liveMode {
		return "https://api-m.paypal.com"
	} else {
		return "https://api-m.sandbox.paypal.com"
	}
}
