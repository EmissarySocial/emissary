package build

import (
	"net/http"
	"testing"
	"testing/fstest"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// stubDomainFactory satisfies the (62-method) Factory interface by embedding it (nil) and
// overriding only Domain() and Theme(), which is all that ThemeData() reads.  Any other
// Factory call would panic, which is fine: these tests never make one.
type stubDomainFactory struct {
	Factory
	domainService *service.Domain
	themeService  *service.Theme
}

// Domain returns the Domain service. Implements the Factory interface.
func (f stubDomainFactory) Domain() *service.Domain { return f.domainService }

// Theme returns the Theme service. Implements the Factory interface.
func (f stubDomainFactory) Theme() *service.Theme { return f.themeService }

// newTestDomainFactory returns a Factory whose cached Domain record carries the provided
// theme data and (secret) operational data, and whose Theme declares a "showNavigation"
// flag that defaults to TRUE -- the shape ThemeData falls back to for an unsaved setting.
func newTestDomainFactory(themeData mapof.Any, data mapof.String) stubDomainFactory {

	domainService := service.NewDomain()

	// Get() hands back a pointer INTO the service's cached record, so writing through it
	// is the only way to seed one without a database.
	domain := domainService.Get()
	domain.ThemeData = themeData
	domain.Data = data
	domain.ThemeID = "test"

	themeService := service.NewTheme(nil, nil, nil)

	// A theme folder carrying nothing but its definition: no HTML, bundles, or content,
	// which are the parts Add() skips when the filesystem does not provide them.
	filesystem := fstest.MapFS{
		"theme.hjson": &fstest.MapFile{Data: []byte(`{
			themeId: test
			schema: {type:"object", properties:{
				themeData: {type:"object", properties:{
					"showNavigation": {type:"boolean", default:true}
					"stylesheet": {type:"string"}
				}}
			}}
		}`)},
	}

	if err := themeService.Add("test", filesystem, filesystem["theme.hjson"].Data); err != nil {
		panic(err)
	}

	return stubDomainFactory{domainService: &domainService, themeService: &themeService}
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

// TestCommon_ThemeDataFallsBackToSchemaDefault pins the fallback that makes a
// "visible by default" setting behave the way its form toggle displays.  A Domain begins
// with NO themeData keys at all, so a setting the owner has never saved has no stored
// value -- and reading that as the empty string would hide a navigation bar that the
// settings form is simultaneously showing as ON.
func TestCommon_ThemeDataFallsBackToSchemaDefault(t *testing.T) {

	builder := Common{_factory: newTestDomainFactory(mapof.NewAny(), mapof.NewString())}

	// Never saved: the Theme's declared default answers
	require.Equal(t, "true", builder.ThemeData("showNavigation"))

	// Declared with no default, and undeclared entirely, both stay empty
	require.Equal(t, "", builder.ThemeData("stylesheet"))
	require.Equal(t, "", builder.ThemeData("undeclared-token"))
}

// TestCommon_ThemeDataStoredValueWinsOverDefault verifies that the default only fills in for
// an ABSENT value.  Once the owner saves the toggle OFF, that FALSE has to survive -- if the
// default won here, the setting could never be turned off at all.
func TestCommon_ThemeDataStoredValueWinsOverDefault(t *testing.T) {

	builder := Common{_factory: newTestDomainFactory(
		mapof.Any{"showNavigation": false},
		mapof.NewString(),
	)}

	require.Equal(t, "false", builder.ThemeData("showNavigation"))
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
