package consumer

import (
	"context"
	"testing"
	"time"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/turbine/queue"
	"github.com/stretchr/testify/require"
)

// fakeTransaction returns a transaction function that runs the callback once, and then
// reports commitErr if the callback itself succeeded
func fakeTransaction(commitErr error) func(context.Context, data.TransactionCallbackFunc) (any, error) {

	return func(_ context.Context, callback data.TransactionCallbackFunc) (any, error) {

		result, err := callback(emptySession{})

		if err != nil {
			return result, err
		}

		return result, commitErr
	}
}

// TestWithTransactionResult_FailureStaysAFailure confirms a permanent failure is not retried
func TestWithTransactionResult_FailureStaysAFailure(t *testing.T) {

	failure := derp.BadRequest("test", "Mailchimp refused this member")

	result := withTransactionResult(fakeTransaction(nil), func(data.Session) queue.Result {
		return queue.Failure(failure)
	})

	require.Equal(t, queue.ResultStatusFailure, result.Status)
	require.Equal(t, failure, result.Error)
}

// TestWithTransactionResult_ErrorStaysAnError confirms a retryable error is still retried
func TestWithTransactionResult_ErrorStaysAnError(t *testing.T) {

	transient := derp.BadGateway("test", "Mailchimp is unreachable")

	result := withTransactionResult(fakeTransaction(nil), func(data.Session) queue.Result {
		return queue.Error(transient)
	})

	require.Equal(t, queue.ResultStatusError, result.Status)
	require.Equal(t, transient, result.Error)
}

// TestWithTransactionResult_SuccessAndRequeuePassThrough confirms results without an error are untouched
func TestWithTransactionResult_SuccessAndRequeuePassThrough(t *testing.T) {

	success := withTransactionResult(fakeTransaction(nil), func(data.Session) queue.Result {
		return queue.Success()
	})

	require.Equal(t, queue.ResultStatusSuccess, success.Status)
	require.Nil(t, success.Error)

	requeue := withTransactionResult(fakeTransaction(nil), func(data.Session) queue.Result {
		return queue.Requeue(time.Minute)
	})

	require.Equal(t, queue.ResultStatusRequeue, requeue.Status)
	require.Equal(t, time.Minute, requeue.Delay)
}

// TestWithTransactionResult_FailedCommitIsRetried confirms a success the database did not keep is retried
func TestWithTransactionResult_FailedCommitIsRetried(t *testing.T) {

	result := withTransactionResult(fakeTransaction(derp.Internal("test", "Commit failed")), func(data.Session) queue.Result {
		return queue.Success()
	})

	require.Equal(t, queue.ResultStatusError, result.Status)
	require.Error(t, result.Error)
}

// TestWithTransactionResult_UnstartedTransactionIsRetried confirms a transaction that never ran the handler is retried
func TestWithTransactionResult_UnstartedTransactionIsRetried(t *testing.T) {

	unstarted := func(context.Context, data.TransactionCallbackFunc) (any, error) {
		return nil, derp.Internal("test", "Starting database session")
	}

	called := false

	result := withTransactionResult(unstarted, func(data.Session) queue.Result {
		called = true
		return queue.Success()
	})

	require.False(t, called)
	require.Equal(t, queue.ResultStatusError, result.Status)
	require.Error(t, result.Error)
}

// TestWithTransactionResult_SilentUnstartedTransactionStillNamesItself confirms the retry carries a real error
func TestWithTransactionResult_SilentUnstartedTransactionStillNamesItself(t *testing.T) {

	silent := func(context.Context, data.TransactionCallbackFunc) (any, error) {
		return nil, nil
	}

	result := withTransactionResult(silent, func(data.Session) queue.Result {
		return queue.Success()
	})

	require.Equal(t, queue.ResultStatusError, result.Status)
	require.NotZero(t, derp.ErrorCode(result.Error), "a code-0 error is the derp.Wrap(nil) signature")
}

// TestWithTransactionResult_LastAttemptWins confirms a callback the driver retries reports its final attempt
func TestWithTransactionResult_LastAttemptWins(t *testing.T) {

	// The mongo driver re-runs the callback on a transient transaction error
	retrying := func(_ context.Context, callback data.TransactionCallbackFunc) (any, error) {
		_, _ = callback(emptySession{})
		return callback(emptySession{})
	}

	attempt := 0

	result := withTransactionResult(retrying, func(data.Session) queue.Result {
		attempt++

		if attempt == 1 {
			return queue.Error(derp.Internal("test", "Transient"))
		}

		return queue.Success()
	})

	require.Equal(t, 2, attempt)
	require.Equal(t, queue.ResultStatusSuccess, result.Status)
	require.Nil(t, result.Error)
}
