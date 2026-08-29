package build

import (
	"net/http"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// stubDomainFactory satisfies the (62-method) Factory interface by embedding it (nil) and
// overriding only Domain(), which is all that ThemeData() reads.  Any other Factory call
// would panic, which is fine: these tests never make one.
type stubDomainFactory struct {
	Factory
	domainService *service.Domain
}

// Domain returns the Domain service. Implements the Factory interface.
func (f stubDomainFactory) Domain() *service.Domain { return f.domainService }

// newTestDomainFactory returns a Factory whose cached Domain record carries the provided
// theme data and (secret) operational data.
func newTestDomainFactory(themeData mapof.Any, data mapof.String) stubDomainFactory {

	domainService := service.NewDomain()

	// Get() hands back a pointer INTO the service's cached record, so writing through it
	// is the only way to seed one without a database.
	domain := domainService.Get()
	domain.ThemeData = themeData
	domain.Data = data

	return stubDomainFactory{domainService: &domainService}
}

// TestReportedBug_ThemeDataReadsTheDomainRecord is the regression test for a custom
// stylesheet that saved correctly but never appeared on any page: Common.ThemeData read
// model.Theme.Data -- the static map parsed from theme.hjson, which the admin form never
// writes and which every Domain on the server shares -- instead of the Domain record.
func TestReportedBug_ThemeDataReadsTheDomainRecord(t *testing.T) {

	factory := newTestDomainFactory(
		mapof.Any{"stylesheet": "body { color: red; }"},
		mapof.NewString(),
	)

	builder := Common{_factory: factory}

	require.Equal(t, "body { color: red; }", builder.ThemeData("stylesheet"))
	require.Equal(t, "", builder.ThemeData("missing-token"))
}

// TestCommon_ThemeDataNeverReadsSecrets pins the separation between the two maps on a
// Domain: ThemeData is rendered into public pages, while Data holds operational secrets
// such as the VAPID private key.  Merging them would publish the secrets.
func TestCommon_ThemeDataNeverReadsSecrets(t *testing.T) {

	factory := newTestDomainFactory(
		mapof.NewAny(),
		mapof.String{"vapidPrivateKey": "SECRET"},
	)

	builder := Common{_factory: factory}

	require.Equal(t, "", builder.ThemeData("vapidPrivateKey"))
}

// TestTheme_IsIndexable verifies that the domain-level pages opt out of search
// indexing.  Common defaults to TRUE, so a missing override would quietly publish every
// sign-in and password-reset page.
func TestTheme_IsIndexable(t *testing.T) {

	require.True(t, Common{}.IsIndexable(), "Common must default to indexable")
	require.False(t, Theme{}.IsIndexable())
	require.False(t, PasswordReset{}.IsIndexable())
	require.False(t, OAuthAuthorization{}.IsIndexable())
}

// TestPasswordReset_Accessors verifies that the reset-code page reports the User that was
// actually loaded, and reads the one-time code from the request URL.
func TestPasswordReset_Accessors(t *testing.T) {

	userID, err := primitive.ObjectIDFromHex("123456781234567812345678")
	require.Nil(t, err)

	user := model.NewUser()
	user.UserID = userID
	user.Username = "sarah"

	request, err := http.NewRequest(http.MethodGet, "https://example.com/signin/reset-code?userId=sarah&code=ABC123", nil)
	require.Nil(t, err)

	builder := PasswordReset{
		_user: user,
		Theme: Theme{Common: Common{_request: request}},
	}

	// RULE: The reset link addresses the User by username here, so UserID must report the
	// loaded record's ID -- not the URL value -- or the form posts back an unverified name.
	require.Equal(t, "123456781234567812345678", builder.UserID())
	require.Equal(t, "sarah", builder.Username())
	require.Equal(t, "ABC123", builder.Code())
}
