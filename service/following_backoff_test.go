package service

import (
	"context"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/realtime"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
)

/******************************************
 * In-Memory Fakes
 ******************************************/

// backoffCollection is a write-only data.Collection that keeps every record handed to Save
type backoffCollection struct {
	saved []model.Following
}

// Context implements the data.Collection interface, returning a background context
func (c *backoffCollection) Context() context.Context { return context.Background() }

// Save records the Following that the service wrote. Implements the data.Collection interface.
func (c *backoffCollection) Save(object data.Object, _ string) error {

	following, ok := object.(*model.Following)

	if !ok {
		return derp.Internal("test", "unexpected target type")
	}

	c.saved = append(c.saved, *following)
	return nil
}

// Count implements the data.Collection interface. Unused by these tests.
func (c *backoffCollection) Count(exp.Expression, ...option.Option) (int64, error) {
	return 0, derp.Internal("test", "unused")
}

// Query implements the data.Collection interface. Unused by these tests.
func (c *backoffCollection) Query(any, exp.Expression, ...option.Option) error {
	return derp.Internal("test", "unused")
}

// Iterator implements the data.Collection interface. Unused by these tests.
func (c *backoffCollection) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.Internal("test", "unused")
}

// Load implements the data.Collection interface. Unused by these tests.
func (c *backoffCollection) Load(exp.Expression, data.Object, ...option.Option) error {
	return derp.Internal("test", "unused")
}

// Delete implements the data.Collection interface. Unused by these tests.
func (c *backoffCollection) Delete(data.Object, string) error {
	return derp.Internal("test", "unused")
}

// HardDelete implements the data.Collection interface. Unused by these tests.
func (c *backoffCollection) HardDelete(exp.Expression) error {
	return derp.Internal("test", "unused")
}

// backoffSession hands out a single shared backoffCollection
type backoffSession struct {
	collection *backoffCollection
}

// Collection implements the data.Session interface, returning this stub's single collection
func (s backoffSession) Collection(string) data.Collection { return s.collection }

// Context implements the data.Session interface, returning a background context
func (s backoffSession) Context() context.Context { return context.Background() }

// Close implements the data.Session interface. The stub holds no resources to release.
func (s backoffSession) Close() {}

/******************************************
 * Tests
 ******************************************/

// TestFollowing_SetStatusFailure_SchedulesTheBackoff confirms that the wait is applied to the
// record, counted from the failure that was just recorded
func TestFollowing_SetStatusFailure_SchedulesTheBackoff(t *testing.T) {

	collection := &backoffCollection{}
	session := backoffSession{collection: collection}

	// Buffered so the service's own status broadcast does not block this test
	updates := make(chan realtime.Message, 8)
	service := Following{sseUpdateChannel: updates}

	following := model.NewFollowing()
	following.LastPolled = 1000

	// Three consecutive failures, checked as they accumulate
	for _, want := range []time.Duration{1 * time.Minute, 2 * time.Minute, 4 * time.Minute} {

		before := time.Now().Unix()
		require.NoError(t, service.SetStatusFailure(session, &following, "Feed is gone"))
		after := time.Now().Unix()

		require.Equal(t, model.FollowingStatusFailure, following.Status)
		require.Equal(t, "Feed is gone", following.StatusMessage)

		// The wait is counted from now, so the expected value is a range of one second
		require.GreaterOrEqual(t, following.NextPoll, before+int64(want.Seconds()))
		require.LessOrEqual(t, following.NextPoll, after+int64(want.Seconds()))

		// RULE: LastPolled records the last SUCCESS, so a failure must leave it alone
		require.Equal(t, int64(1000), following.LastPolled)
	}

	require.Equal(t, 3, following.ErrorCount)
	require.Len(t, collection.saved, 3)
}
