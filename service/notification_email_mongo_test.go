package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// fakeNotificationEmailer records the unread emails it is asked to send
type fakeNotificationEmailer struct {
	lock         sync.Mutex
	unconfigured bool
	sendError    error
	sent         [][]model.Notification
	moreCounts   []int64
}

// IsConfigured implements notificationEmailer
func (emailer *fakeNotificationEmailer) IsConfigured() bool {
	return !emailer.unconfigured
}

// SendUnreadNotifications implements notificationEmailer
func (emailer *fakeNotificationEmailer) SendUnreadNotifications(_ *model.User, notifications []model.Notification, moreCount int64) error {

	emailer.lock.Lock()
	defer emailer.lock.Unlock()

	if emailer.sendError != nil {
		return emailer.sendError
	}

	emailer.sent = append(emailer.sent, notifications)
	emailer.moreCounts = append(emailer.moreCounts, moreCount)
	return nil
}

// sentCount returns the number of emails sent so far
func (emailer *fakeNotificationEmailer) sentCount() int {
	emailer.lock.Lock()
	defer emailer.lock.Unlock()
	return len(emailer.sent)
}

// TestNotification_EmailUnread_SendsOnce pins that an owed User gets one email, and that the
// watermark then keeps the same Notifications from causing another
func TestNotification_EmailUnread_SendsOnce(t *testing.T) {

	_, session := newReplicaSetSession(t)
	service, emailer := newEmailTestService()
	user := saveEmailTestUser(t, session)

	older := saveUnread(t, session, user.UserID, model.NotificationTypeLike)
	time.Sleep(2 * time.Millisecond)
	newer := saveUnread(t, session, user.UserID, model.NotificationTypeFollow)

	sent, err := service.EmailUnread(session, &user, hoursFromNow(25))
	require.NoError(t, err)
	require.True(t, sent)

	require.Len(t, emailer.sent, 1)
	require.Len(t, emailer.sent[0], 2)
	require.Equal(t, newer.NotificationID, emailer.sent[0][0].NotificationID, "the newest row comes first")
	require.Equal(t, older.NotificationID, emailer.sent[0][1].NotificationID)
	require.Zero(t, emailer.moreCounts[0])

	// The watermark moves to the newest unread Notification, in the database and on the User
	stored := storedWatermark(t, session, user.UserID)
	require.Equal(t, emailer.sent[0][0].CreateDate, stored)
	require.Equal(t, stored, user.NotificationEmailDate)

	// The same unread rows cause no second email, however long they stay unread
	sent, err = service.EmailUnread(session, &user, hoursFromNow(100))
	require.NoError(t, err)
	require.False(t, sent)
	require.Len(t, emailer.sent, 1)
}

// TestNotification_EmailUnread_NotOwed pins the cases that send nothing and claim nothing
func TestNotification_EmailUnread_NotOwed(t *testing.T) {

	_, session := newReplicaSetSession(t)

	test := func(name string, hoursAhead int, setup func(*model.User, *fakeNotificationEmailer, *Notification)) {
		t.Run(name, func(t *testing.T) {
			service, emailer := newEmailTestService()
			user := saveEmailTestUser(t, session)
			saveUnread(t, session, user.UserID, model.NotificationTypeLike)
			setup(&user, emailer, service)

			sent, err := service.EmailUnread(session, &user, hoursFromNow(hoursAhead))
			require.NoError(t, err)
			require.False(t, sent)
			require.Zero(t, emailer.sentCount())
			require.Zero(t, storedWatermark(t, session, user.UserID), "nothing was claimed")
		})
	}

	test("the Notification is younger than the span", 23, func(*model.User, *fakeNotificationEmailer, *Notification) {})
	test("the User chose No email", 100, func(user *model.User, _ *fakeNotificationEmailer, _ *Notification) { user.NotificationEmailHours = 0 })
	test("the User has no email address", 100, func(user *model.User, _ *fakeNotificationEmailer, _ *Notification) { user.EmailAddress = "" })
	test("the Domain cannot send email", 100, func(_ *model.User, emailer *fakeNotificationEmailer, _ *Notification) { emailer.unconfigured = true })

	test("the Notification was read", 100, func(user *model.User, _ *fakeNotificationEmailer, service *Notification) {
		require.NoError(t, service.MarkAllRead(session, user.UserID, 1234))
	})
}

// TestNotification_EmailUnread_ShorterSpan pins that the User's own span decides
func TestNotification_EmailUnread_ShorterSpan(t *testing.T) {

	_, session := newReplicaSetSession(t)
	service, emailer := newEmailTestService()
	user := saveEmailTestUser(t, session)
	user.NotificationEmailHours = 1
	saveUnread(t, session, user.UserID, model.NotificationTypeLike)

	sent, err := service.EmailUnread(session, &user, hoursFromNow(2))
	require.NoError(t, err)
	require.True(t, sent)
	require.Equal(t, 1, emailer.sentCount())
}

// TestNotification_EmailUnread_HiddenActor pins that a Notification from an actor the User now
// blocks is left out, and that an email with nothing left is not sent while the watermark moves
func TestNotification_EmailUnread_HiddenActor(t *testing.T) {

	_, session := newReplicaSetSession(t)
	service, emailer := newEmailTestService()
	user := saveEmailTestUser(t, session)

	blocked := model.NewNotification()
	blocked.UserID = user.UserID
	blocked.Type = model.NotificationTypeMention
	blocked.Actor.ProfileURL = "https://evil.example/@spammer"
	require.NoError(t, session.Collection("Notification").Save(&blocked, "test setup"))

	saveBlockRule(t, session, user.UserID, "https://evil.example/@spammer")

	sent, err := service.EmailUnread(session, &user, hoursFromNow(25))
	require.NoError(t, err)
	require.False(t, sent)
	require.Zero(t, emailer.sentCount())
	require.Equal(t, storedWatermark(t, session, user.UserID), user.NotificationEmailDate)
	require.NotZero(t, user.NotificationEmailDate, "the watermark still moves, so the hidden row is not asked about again")

	// A later Notification from someone else is sent, without the hidden one
	time.Sleep(2 * time.Millisecond)
	friendly := saveUnread(t, session, user.UserID, model.NotificationTypeFollow)

	sent, err = service.EmailUnread(session, &user, hoursFromNow(50))
	require.NoError(t, err)
	require.True(t, sent)
	require.Len(t, emailer.sent[0], 1)
	require.Equal(t, friendly.NotificationID, emailer.sent[0][0].NotificationID)
}

// saveBlockRule stores a Rule that blocks one actor for one User
func saveBlockRule(t *testing.T, session data.Session, userID primitive.ObjectID, actorURL string) {

	t.Helper()

	rule := model.NewRule()
	rule.UserID = userID
	rule.Type = model.RuleTypeActor
	rule.Action = model.RuleActionBlock
	rule.Trigger = actorURL
	rule.MatchKey = model.RuleMatchKey(rule.Type, rule.Trigger)

	require.NoError(t, session.Collection("Rule").Save(&rule, "test setup"))
}

// TestNotification_EmailUnread_FailedSend pins that a failed send puts the watermark back, and
// that the error does not carry the User's email address
func TestNotification_EmailUnread_FailedSend(t *testing.T) {

	_, session := newReplicaSetSession(t)
	service, emailer := newEmailTestService()
	user := saveEmailTestUser(t, session)
	saveUnread(t, session, user.UserID, model.NotificationTypeLike)

	emailer.sendError = errors.New("smtp is down")

	sent, err := service.EmailUnread(session, &user, hoursFromNow(25))
	require.Error(t, err)
	require.False(t, sent)
	require.Zero(t, storedWatermark(t, session, user.UserID))
	require.Zero(t, user.NotificationEmailDate)
	require.NotContains(t, err.Error(), user.EmailAddress)

	// The next run sends it
	emailer.sendError = nil

	sent, err = service.EmailUnread(session, &user, hoursFromNow(26))
	require.NoError(t, err)
	require.True(t, sent)
}

// storedWatermark reads a User's notification email watermark back from the database
func storedWatermark(t *testing.T, session data.Session, userID primitive.ObjectID) int64 {

	t.Helper()

	user := model.NewUser()
	userService := User{}
	require.NoError(t, userService.LoadByID(session, userID, &user))

	return user.NotificationEmailDate
}

// TestNotification_EmailUnread_PageOfSixty pins that the email lists 60 rows and counts the rest
func TestNotification_EmailUnread_PageOfSixty(t *testing.T) {

	_, session := newReplicaSetSession(t)
	service, emailer := newEmailTestService()
	user := saveEmailTestUser(t, session)

	for range notificationEmailPageSize + 5 {
		saveUnread(t, session, user.UserID, model.NotificationTypeLike)
	}

	sent, err := service.EmailUnread(session, &user, hoursFromNow(25))
	require.NoError(t, err)
	require.True(t, sent)
	require.Len(t, emailer.sent[0], notificationEmailPageSize)
	require.Equal(t, int64(5), emailer.moreCounts[0])
}

// TestNotification_EmailUnread_TwoServers pins that two servers running the hourly task at once
// send one email between them, because the database decides which claim wins
func TestNotification_EmailUnread_TwoServers(t *testing.T) {

	server, session := newReplicaSetSession(t)

	for round := range 20 {

		service, emailer := newEmailTestService()
		user := saveEmailTestUser(t, session)
		saveUnread(t, session, user.UserID, model.NotificationTypeLike)

		now := hoursFromNow(25)
		start := make(chan struct{})
		var group sync.WaitGroup

		for range 2 {
			group.Go(func() {

				// Each server has its own session and its own copy of the User
				ownSession, err := server.Session(context.Background())
				if err != nil {
					t.Errorf("round %d: opening session: %v", round, err)
					return
				}
				defer ownSession.Close()

				ownUser := user
				<-start

				if _, err := service.EmailUnread(ownSession, &ownUser, now); err != nil {
					t.Errorf("round %d: %v", round, err)
				}
			})
		}

		close(start)
		group.Wait()

		require.Equal(t, 1, emailer.sentCount(), "round %d", round)
	}
}

// newEmailTestService returns a Notification service wired to a fake emailer and a real Rule service
func newEmailTestService() (*Notification, *fakeNotificationEmailer) {
	ruleService := NewRule()
	emailer := &fakeNotificationEmailer{}
	return &Notification{ruleService: &ruleService, emailService: emailer}, emailer
}

// saveEmailTestUser stores a User who has an email address and the default span of 24 hours
func saveEmailTestUser(t *testing.T, session data.Session) model.User {

	t.Helper()

	user := model.NewUser()
	user.EmailAddress = "sarah@connor.mil"
	require.NoError(t, session.Collection("User").Save(&user, "test setup"))

	return user
}

// hoursFromNow returns a moment that many hours ahead, in Unix epoch MILLISECONDS.  The tests save
// their Notifications now, and age them by moving the clock that EmailUnread is given.
func hoursFromNow(hours int) int64 {
	return time.Now().Add(time.Duration(hours) * time.Hour).UnixMilli()
}
