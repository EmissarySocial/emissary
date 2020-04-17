package model

// AlertTypeEmail defines the database constant for a Person's email address
const AlertTypeEmail = "EMAIL"

// AlertTypeSMS defines the database constant for a Person's SMS phone number
const AlertTypeSMS = "SMS"

// AlertTypeSlack defines the database constant for a Person's Slack ID
const AlertTypeSlack = "SLACK"

// AlertFrequencyInstant defines the database constant for sending instant alerts to a Person
const AlertFrequencyInstant = "INSTANT"

// AlertFrequencyHourly defines the database constant for sending hourly alerts to a Person
const AlertFrequencyHourly = "HOURLY"

// AlertFrequencyDaily defines the database constant for sending daily alerts to a Person
const AlertFrequencyDaily = "DAILY"

// AlertFrequencyNone defines the database constant for sending NO alerts to a Person
const AlertFrequencyNone = "NONE"

// AlertType defines the settings for sending an alert to a Person via a specific technology
type AlertType struct {
	Address   string
	Frequency string
}
