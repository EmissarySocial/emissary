package model

// UserMapIDStripe identifies the Stripe CustomerID for a User
// https://docs.stripe.com/api/customers/object#customer_object-id
const UserMapIDStripe = "STRIPE"

// NotificationEmailHoursDefault is the number of hours a new User's notifications stay unread
// before they are emailed
const NotificationEmailHoursDefault = 24

// millisecondsPerHour converts a span in hours into the unit of journal.createDate
const millisecondsPerHour = int64(60 * 60 * 1000)

// NotificationEmailHours returns every value that User.NotificationEmailHours may hold.  Zero
// means that no email is sent.
func NotificationEmailHours() []int {
	return []int{0, 1, 6, 12, 24, 48}
}
