package build

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

// StepReadForm copies approved form fields into the Builder's temporary scope and NEVER into the
// object being built -- which, for a Stream, is the page record itself. Post touches only
// request() and setString(), so a minimal stub builder is enough.

// stubFormBuilder is a build.Builder exposing the two methods the form steps reach
type stubFormBuilder struct {
	Builder
	req    *http.Request
	values mapof.String
}

// request implements the Builder interface, returning this stub's request
func (b stubFormBuilder) request() *http.Request { return b.req }

// setString implements the Builder interface, recording what the step published
func (b stubFormBuilder) setString(name string, value string) { b.values[name] = value }

// newStubFormBuilder returns a stub carrying a urlencoded POST of the provided fields
func newStubFormBuilder(fields url.Values) stubFormBuilder {
	return newStubFormBuilderWithQuery("", fields)
}

// newStubFormBuilderWithQuery returns a stub whose POST also carries a URL query string
func newStubFormBuilderWithQuery(query string, fields url.Values) stubFormBuilder {

	request := httptest.NewRequest(http.MethodPost, "/?"+query, strings.NewReader(fields.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return stubFormBuilder{req: request, values: make(mapof.String)}
}

// behaviorError applies a PipelineBehavior to a fresh result and returns the error it recorded
func behaviorError(behavior PipelineBehavior) error {

	result := NewPipelineResult()
	behavior(&result)

	return result.Error
}

// contactFormSchema mirrors the shape a contact form declares on its read-form step
func contactFormSchema() schema.Schema {
	return schema.New(schema.Object{
		Properties: schema.ElementMap{
			"name":    schema.String{MaxLength: 32, Required: true},
			"email":   schema.String{Format: "email", MaxLength: 255, Required: true},
			"message": schema.String{MaxLength: 64, Required: true},
		},
	})
}

// TestStepReadForm verifies that declared fields reach the Builder's temporary scope
func TestStepReadForm(t *testing.T) {

	builder := newStubFormBuilder(url.Values{
		"name":    {"Sarah"},
		"email":   {"sarah@example.com"},
		"message": {"Hello there"},
	})

	result := StepReadForm{Schema: contactFormSchema()}.Post(builder, nil)

	require.Nil(t, result)
	require.Equal(t, "Sarah", builder.values["name"])
	require.Equal(t, "sarah@example.com", builder.values["email"])
	require.Equal(t, "Hello there", builder.values["message"])
}

// TestStepReadForm_IgnoresUndeclaredFields verifies that a field the schema does not declare is
// never read. The schema is the allowlist; anything else a visitor posts is dropped on the floor.
func TestStepReadForm_IgnoresUndeclaredFields(t *testing.T) {

	builder := newStubFormBuilder(url.Values{
		"name":    {"Sarah"},
		"email":   {"sarah@example.com"},
		"message": {"Hello"},
		"token":   {"../../etc/passwd"},
	})

	require.Nil(t, StepReadForm{Schema: contactFormSchema()}.Post(builder, nil))
	require.NotContains(t, builder.values, "token")
}

// TestStepReadForm_RejectsOverLongValue verifies that an oversized field HALTS rather than being
// silently shortened. schema.Set truncates to maxLength and reports success, which would deliver
// half of a visitor's message while telling them it sent whole (D10).
func TestStepReadForm_RejectsOverLongValue(t *testing.T) {

	builder := newStubFormBuilder(url.Values{
		"name":    {"Sarah"},
		"email":   {"sarah@example.com"},
		"message": {strings.Repeat("x", 65)},
	})

	result := StepReadForm{Schema: contactFormSchema()}.Post(builder, nil)

	require.NotNil(t, result)
	require.Error(t, behaviorError(result))
	require.Empty(t, builder.values, "nothing may be published when the form is rejected")
}

// TestStepReadForm_LengthIsCountedInRunes verifies that maxLength bounds CHARACTERS, not bytes.
// Counting bytes would make the limit 2-4x stricter than the schema advertises for any non-Latin
// script or emoji, rejecting a message that fits -- and it would disagree with schema.Set, which
// truncates by runes. Every other length limit in this codebase counts runes; so does this one.
func TestStepReadForm_LengthIsCountedInRunes(t *testing.T) {

	// 64 characters of Japanese: within the schema's 64-rune bound, but 192 bytes
	message := strings.Repeat("日", 64)

	require.Equal(t, 64, utf8.RuneCountInString(message))
	require.Greater(t, len(message), 64, "the test subject must be longer in bytes than in runes")

	builder := newStubFormBuilder(url.Values{
		"name":    {"Sarah"},
		"email":   {"sarah@example.com"},
		"message": {message},
	})

	result := StepReadForm{Schema: contactFormSchema()}.Post(builder, nil)

	require.Nil(t, result, "a message that fits in runes must be accepted")
	require.Equal(t, message, builder.values["message"], "the message must survive intact")
}

// TestStepReadForm_IgnoresQueryString verifies that a declared field is read from the request
// BODY only. formdata.Parse merges the query string and returns both values under one name,
// which this step joins with a comma -- so without ParseBody, the link
// "/contact/submit?message=..." would append the attacker's text to whatever the visitor wrote,
// and the email would go out carrying it. The visitor would see only their own words.
func TestStepReadForm_IgnoresQueryString(t *testing.T) {

	builder := newStubFormBuilderWithQuery("message=INJECTED", url.Values{
		"name":    {"Sarah"},
		"email":   {"sarah@example.com"},
		"message": {"Hello"},
	})

	result := StepReadForm{Schema: contactFormSchema()}.Post(builder, nil)

	require.Nil(t, result)
	require.Equal(t, "Hello", builder.values["message"], "the query value must not reach the email")
	require.NotContains(t, builder.values["message"], "INJECTED")
}

// TestStepReadForm_QueryCannotSupplyAMissingField verifies that the query string cannot stand in
// for a required field the visitor never filled out. Reading it would let a crafted link forge
// the whole submission through a bare GET-shaped POST.
func TestStepReadForm_QueryCannotSupplyAMissingField(t *testing.T) {

	builder := newStubFormBuilderWithQuery("message=SuppliedByLink", url.Values{
		"name":  {"Sarah"},
		"email": {"sarah@example.com"},
	})

	result := StepReadForm{Schema: contactFormSchema()}.Post(builder, nil)

	require.NotNil(t, result)
	require.Error(t, behaviorError(result), "a required field must not be satisfiable from the query string")
}

// TestStepReadForm_RejectsMissingRequiredField verifies that a required field left out fails
// here, rather than rendering as an empty value inside an email
func TestStepReadForm_RejectsMissingRequiredField(t *testing.T) {

	builder := newStubFormBuilder(url.Values{"name": {"Sarah"}, "email": {"sarah@example.com"}})

	result := StepReadForm{Schema: contactFormSchema()}.Post(builder, nil)

	require.NotNil(t, result)
	require.Error(t, behaviorError(result))
}

// TestStepReadForm_RejectsInvalidEmail verifies that the schema's format check is enforced.
// The visitor's address becomes Reply-To, where an invalid value fails the whole send.
func TestStepReadForm_RejectsInvalidEmail(t *testing.T) {

	builder := newStubFormBuilder(url.Values{
		"name":    {"Sarah"},
		"email":   {"not-an-email"},
		"message": {"Hello"},
	})

	result := StepReadForm{Schema: contactFormSchema()}.Post(builder, nil)

	require.NotNil(t, result)
	require.Error(t, behaviorError(result))
}
