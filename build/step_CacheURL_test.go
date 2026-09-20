package build

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/headers"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// newCacheURLBuilder assembles the smallest Stream builder StepCacheURL can run against: a Stream
// carrying a revision, and an anonymous GET that may or may not present a cache validator.
func newCacheURLBuilder(t *testing.T, revision int64, ifNoneMatch string) Stream {

	t.Helper()

	return newCacheURLBuilderAs(t, revision, ifNoneMatch, model.Authorization{})
}

// newCacheURLBuilderAs is newCacheURLBuilder for a request that carries the provided Authorization.
func newCacheURLBuilderAs(t *testing.T, revision int64, ifNoneMatch string, authorization model.Authorization) Stream {

	t.Helper()

	stream := model.NewStream()
	stream.Revision = revision

	request := httptest.NewRequest(http.MethodGet, "/000000000000000000000000", nil)

	if ifNoneMatch != "" {
		request.Header.Set("If-None-Match", ifNoneMatch)
	}

	return Stream{
		_stream: &stream,
		CommonWithTemplate: CommonWithTemplate{
			Common: Common{
				_request:       request,
				_authorization: authorization,
			},
		},
	}
}

// TestStepCacheURL_PublishesTheTagTheClientKeeps is the regression test for the defect that made the
// 304 below unreachable: this step published the bare revision while StepViewHTML published the
// variant-tagged form, and the pipeline applies StepViewHTML's headers last -- so the value the
// client echoed in If-None-Match could never equal the value this step compared against.
func TestStepCacheURL_PublishesTheTagTheClientKeeps(t *testing.T) {

	builder := newCacheURLBuilder(t, 7, "")

	result := applyBehavior(StepCacheURL{}.Get(builder, nil))

	// The expression on the right is exactly what StepViewHTML publishes (step_ViewHTML.go)
	require.Equal(t, headers.ETag(headers.VariantHTML, builder.object()), result.Headers["ETag"])

	// ...which is NOT the bare revision
	require.Equal(t, "7", builder.object().ETag())
	require.Equal(t, `W/"7-html"`, result.Headers["ETag"])
}

// TestStepCacheURL_NotModified answers a matching validator with a bodyless 304
func TestStepCacheURL_NotModified(t *testing.T) {

	builder := newCacheURLBuilder(t, 7, `W/"7-html"`)

	result := applyBehavior(StepCacheURL{CacheControl: "public, max-age=3600"}.Get(builder, nil))

	require.True(t, result.Halt)
	require.Equal(t, http.StatusNotModified, result.GetStatusCode())

	// RULE: AsFullPage is what routes AsHTML to the empty body.  Without it a 304 renders the
	// whole site theme and answers 200.
	require.True(t, result.FullPage)

	// RULE: A 304 repeats the headers its 200 would have carried
	require.Equal(t, `W/"7-html"`, result.Headers["ETag"])
	require.Equal(t, "public, max-age=3600", result.Headers["Cache-Control"])
}

// TestStepCacheURL_Modified continues the pipeline when the client's validator is stale
func TestStepCacheURL_Modified(t *testing.T) {

	builder := newCacheURLBuilder(t, 8, `W/"7-html"`)

	result := applyBehavior(StepCacheURL{CacheControl: "public, max-age=3600"}.Get(builder, nil))

	require.False(t, result.Halt)
	require.Equal(t, http.StatusOK, result.GetStatusCode())
	require.Equal(t, `W/"8-html"`, result.Headers["ETag"])
	require.Equal(t, "public, max-age=3600", result.Headers["Cache-Control"])
}

// TestStepCacheURL_OtherVariantDoesNotMatch confirms that a peer holding the ActivityStreams tag for
// this revision is not answered 304 for the HTML page, which is why the Variant is in the tag
func TestStepCacheURL_OtherVariantDoesNotMatch(t *testing.T) {

	builder := newCacheURLBuilder(t, 7, `W/"7-as2"`)

	result := applyBehavior(StepCacheURL{}.Get(builder, nil))

	require.False(t, result.Halt)
	require.Equal(t, `W/"7-html"`, result.Headers["ETag"])
}

// TestStepCacheURL_NoCacheControl publishes a validator without inventing a policy
func TestStepCacheURL_NoCacheControl(t *testing.T) {

	builder := newCacheURLBuilder(t, 7, "")

	result := applyBehavior(StepCacheURL{}.Get(builder, nil))

	require.Equal(t, `W/"7-html"`, result.Headers["ETag"])
	require.NotContains(t, result.Headers, "Cache-Control")
}

// TestStepCacheURL_SignedInUserGetsNothingFromThisStep covers the rule that keeps a privileged
// rendering out of a shared cache.  The entity-tag tracks the object's revision only, so it is the
// same tag the public page carries -- this step withholds BOTH the 304 and its cache headers, which
// leaves StepViewHTML's "private, no-cache" standing as the policy for the response.
func TestStepCacheURL_SignedInUserGetsNothingFromThisStep(t *testing.T) {

	authorization := model.Authorization{UserID: primitive.NewObjectID()}
	builder := newCacheURLBuilderAs(t, 7, `W/"7-html"`, authorization)

	result := applyBehavior(StepCacheURL{CacheControl: "public, max-age=3600"}.Get(builder, nil))

	require.False(t, result.Halt)
	require.Equal(t, http.StatusOK, result.GetStatusCode())

	// RULE: "public" must never reach a response built for a signed-in viewer
	require.NotContains(t, result.Headers, "Cache-Control")
	require.NotContains(t, result.Headers, "ETag")
}

// TestStepCacheURL_GuestIdentityGetsNothingFromThisStep applies the same rule to a guest Identity,
// which also sees content an anonymous visitor does not.
func TestStepCacheURL_GuestIdentityGetsNothingFromThisStep(t *testing.T) {

	authorization := model.Authorization{IdentityID: primitive.NewObjectID()}
	builder := newCacheURLBuilderAs(t, 7, `W/"7-html"`, authorization)

	result := applyBehavior(StepCacheURL{CacheControl: "public, max-age=3600"}.Get(builder, nil))

	require.False(t, result.Halt)
	require.Equal(t, http.StatusOK, result.GetStatusCode())
	require.NotContains(t, result.Headers, "Cache-Control")
	require.NotContains(t, result.Headers, "ETag")
}

// TestStepCacheURL_Post does nothing on a POST
func TestStepCacheURL_Post(t *testing.T) {

	builder := newCacheURLBuilder(t, 7, `W/"7-html"`)

	require.Nil(t, StepCacheURL{}.Post(builder, nil))
}
