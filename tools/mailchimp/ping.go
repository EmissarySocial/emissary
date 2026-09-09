package mailchimp

// Ping proves that Mailchimp accepts this Client's credential and data center, and nothing else
func (client Client) Ping() error {

	const location = "tools.mailchimp.Client.Ping"

	// GET /ping is Mailchimp's own health check: it needs a valid key, answers with no account
	// data, and is the lightest call there is that proves a credential works.
	if err := client.get("/ping").Send(); err != nil {
		return describeError(err, location, "Unable to reach Mailchimp")
	}

	// Everything's Chimpy!
	return nil
}
