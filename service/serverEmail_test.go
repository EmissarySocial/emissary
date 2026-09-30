package service

import (
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/EmissarySocial/emissary/model"
	emissarytemplates "github.com/EmissarySocial/emissary/tools/templates"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	mail "github.com/xhit/go-simple-mail/v2"
)

/******************************************
 * Outbound Email Headers
 *
 * These tests pin the header contract for outbound email.
 * Header VALUES are protected by go-simple-mail, which strips
 * CRLF from every one and parses address headers with
 * mail.ParseAddress; the tests below pin those guarantees so
 * that losing them fails here. Header NAMES get no such
 * treatment from the library -- they are written verbatim --
 * so Emissary validates those itself, at load time.
 ******************************************/

// testEmail returns a model.Email whose Headers set holds the provided name/template pairs
func testEmail(t *testing.T, headers map[string]string) model.Email {

	t.Helper()

	email := model.NewEmail("test-email", template.FuncMap{})

	for name, value := range headers {
		headerTemplate, err := email.Headers.New(name).Parse(value)
		require.NoError(t, err)
		email.Headers = headerTemplate
	}

	return email
}

// headerLines returns the header block of a rendered message, split into individual lines
func headerLines(message *mail.Email) []string {
	headers, _, _ := strings.Cut(message.GetMessage(), "\r\n\r\n")
	return strings.Split(headers, "\r\n")
}

/******************************************
 * applyHeaders
 ******************************************/

// TestApplyHeaders verifies that a custom header is rendered and added to the message
func TestApplyHeaders(t *testing.T) {

	email := testEmail(t, map[string]string{
		"Reply-To":        "{{.ReplyTo}}",
		"X-Emissary-Test": "{{.Marker}}",
	})

	message := mail.NewMSG()
	data := mapof.Any{"ReplyTo": "visitor@example.com", "Marker": "hello"}

	require.NoError(t, applyHeaders(message, email, data))
	require.NoError(t, message.GetError())

	rendered := message.GetMessage()
	require.Contains(t, rendered, "visitor@example.com")
	require.Contains(t, rendered, "X-Emissary-Test: hello")
}

// TestApplyHeaders_NoHeaders verifies that an Email declaring no headers is a no-op
func TestApplyHeaders_NoHeaders(t *testing.T) {

	message := mail.NewMSG()

	require.NoError(t, applyHeaders(message, testEmail(t, nil), mapof.Any{}))
	require.NoError(t, message.GetError())
}

// TestApplyHeaders_NilHeaders verifies that an uninitialized Headers template does not panic
func TestApplyHeaders_NilHeaders(t *testing.T) {

	message := mail.NewMSG()

	require.NoError(t, applyHeaders(message, model.Email{}, mapof.Any{}))
	require.NoError(t, message.GetError())
}

// TestApplyHeaders_SkipsEmptyValues verifies that a header rendering empty is omitted instead
// of poisoning the message
func TestApplyHeaders_SkipsEmptyValues(t *testing.T) {

	email := testEmail(t, map[string]string{"Reply-To": "{{.ReplyTo}}"})

	message := mail.NewMSG()

	require.NoError(t, applyHeaders(message, email, mapof.Any{"ReplyTo": ""}))
	require.NoError(t, message.GetError())
	require.NotContains(t, message.GetMessage(), "Reply-To")
}

// TestApplyHeaders_MissingKeyIsRejected verifies that a key absent from the data map fails the
// render outright, including in sub-templates created by Headers.New()
func TestApplyHeaders_MissingKeyIsRejected(t *testing.T) {

	// Without missingkey=error this renders "<no value>", which survives the skip-empty rule and
	// then fails mail.ParseAddress.  text/template keeps the option on the shared template set.

	email := testEmail(t, map[string]string{"Reply-To": "{{.ReplyTo}}"})

	message := mail.NewMSG()

	require.ErrorContains(t, applyHeaders(message, email, mapof.Any{}), "Executing 'headers' template")
}

// TestApplyHeaders_EmptyKeyIsStillAllowed verifies that missingkey=error rejects an ABSENT key,
// not an empty one -- supplying "" remains the way a sender omits an optional header.
func TestApplyHeaders_EmptyKeyIsStillAllowed(t *testing.T) {

	email := testEmail(t, map[string]string{"Reply-To": "{{.ReplyTo}}"})

	message := mail.NewMSG()

	require.NoError(t, applyHeaders(message, email, mapof.Any{"ReplyTo": ""}))
	require.NoError(t, message.GetError())
	require.NotContains(t, message.GetMessage(), "Reply-To")
}

// TestApplyHeaders_InjectionIsNeutralized verifies that CRLF in a rendered header value cannot
// forge a header of its own
func TestApplyHeaders_InjectionIsNeutralized(t *testing.T) {

	// go-simple-mail's encoder (its secureHeader) strips the CRLF, not Emissary, so this
	// fails if a version bump ever removes that guarantee

	email := testEmail(t, map[string]string{"X-Emissary-Test": "{{.Value}}"})

	message := mail.NewMSG()
	data := mapof.Any{"Value": "ok\r\nBcc: attacker@example.com"}

	require.NoError(t, applyHeaders(message, email, data))
	require.NoError(t, message.GetError())

	// The forged header must never appear as a line of its own.  Folded continuation lines
	// always begin with whitespace, so a bare "Bcc:" prefix means the injection succeeded.
	for _, line := range headerLines(message) {
		require.False(t, strings.HasPrefix(line, "Bcc:"), "forged header became its own line: %q", line)
	}

	require.Empty(t, message.GetRecipients())
}

// TestApplyHeaders_AddressInjectionIsRejected verifies that mail.ParseAddress rejects CRLF in an
// address header, rather than delivering to an unintended recipient
func TestApplyHeaders_AddressInjectionIsRejected(t *testing.T) {

	// Like the test above, this pins a guarantee that go-simple-mail makes for us

	email := testEmail(t, map[string]string{"Reply-To": "{{.ReplyTo}}"})

	message := mail.NewMSG()
	data := mapof.Any{"ReplyTo": "visitor@example.com\r\nBcc: attacker@example.com"}

	require.NoError(t, applyHeaders(message, email, data))
	require.Error(t, message.GetError())

	for _, line := range headerLines(message) {
		require.False(t, strings.HasPrefix(line, "Bcc:"), "forged header became its own line: %q", line)
	}
}

/******************************************
 * ServerEmail.Add
 ******************************************/

// testServerEmail returns a bare ServerEmail service with an empty email library
func testServerEmail() ServerEmail {
	return ServerEmail{
		// The real funcMap, because shipped body.html files call helpers such as htmlMinimal
		funcMap: emissarytemplates.FuncMap(nullIconProvider{}),
		emails:  make(map[string]model.Email),
	}
}

// testFilesystem returns the minimum filesystem that an email definition requires
func testFilesystem() fstest.MapFS {
	return fstest.MapFS{
		"body.html": &fstest.MapFile{Data: []byte("<p>Hello</p>")},
	}
}

// TestServerEmailAdd_ParsesHeaders verifies that a valid header definition is loaded
func TestServerEmailAdd_ParsesHeaders(t *testing.T) {

	service := testServerEmail()

	definition := []byte(`{
		emailId: test-email
		model: Stream
		to: "{{.To}}"
		subject: "Hello"
		headers: {"Reply-To": "{{.ReplyTo}}"}
	}`)

	require.NoError(t, service.Add(testFilesystem(), definition))

	email := service.emails["test-email"]
	require.NotNil(t, email.Headers.Lookup("Reply-To"))
}

// TestServerEmailAdd_RejectsInvalidHeaderName verifies that a header name which could break
// out of its own field is rejected at load time
func TestServerEmailAdd_RejectsInvalidHeaderName(t *testing.T) {

	table := []struct {
		name       string
		headerName string
	}{
		{"carriage return and newline", `Reply-To\r\nBcc`},
		{"bare newline", `Reply-To\nBcc`},
		{"embedded colon", `Reply-To:Bcc`},
		{"embedded space", `Reply To`},
		{"empty name", ``},
		{"leading space", ` Reply-To`},
	}

	for _, testCase := range table {
		t.Run(testCase.name, func(t *testing.T) {

			service := testServerEmail()

			definition := []byte(`{
				emailId: test-email
				model: Stream
				to: "{{.To}}"
				subject: "Hello"
				headers: {"` + testCase.headerName + `": "value"}
			}`)

			// Assert on the message: an hjson parse failure would otherwise let this test
			// pass without ever reaching the name validation it is meant to cover.
			require.ErrorContains(t, service.Add(testFilesystem(), definition), "Invalid email header name")
			require.NotContains(t, service.emails, "test-email")
		})
	}
}

// testDefinition returns an email definition with the required fields present, plus whatever
// extra lines a test needs (such as a headers block)
func testDefinition(extra string) []byte {
	return []byte(`{
		emailId: test-email
		model: Stream
		to: "{{.To}}"
		subject: "Hello"
		` + extra + `
	}`)
}

// TestServerEmailAdd_RequiresEmailID verifies that a definition must identify itself
func TestServerEmailAdd_RequiresEmailID(t *testing.T) {

	service := testServerEmail()

	definition := []byte(`{
		model: Stream
		to: "{{.To}}"
		subject: "Hello"
	}`)

	require.ErrorContains(t, service.Add(testFilesystem(), definition), "must include an 'emailId'")
	require.Empty(t, service.emails)
}

// TestServerEmailAdd_RequiresModel verifies that a definition must name the model it belongs to
func TestServerEmailAdd_RequiresModel(t *testing.T) {

	service := testServerEmail()

	definition := []byte(`{
		emailId: test-email
		to: "{{.To}}"
		subject: "Hello"
	}`)

	require.ErrorContains(t, service.Add(testFilesystem(), definition), "must include a 'model'")
	require.Empty(t, service.emails)
}

// TestServerEmailAdd_AcceptsModelOutsideTemplateRegistry verifies that an email may name a model
// the Template registry does not know, such as "Follower"
func TestServerEmailAdd_AcceptsModelOutsideTemplateRegistry(t *testing.T) {

	// An email names the object the message is about, not a builder model (D16 in CONTACT-FORM)

	service := testServerEmail()

	definition := []byte(`{
		emailId: test-email
		model: Follower
		to: "{{.To}}"
		subject: "Hello"
	}`)

	require.NoError(t, service.Add(testFilesystem(), definition))
	require.Contains(t, service.emails, "test-email")
}

/******************************************
 * Model Assertions
 *
 * RequireModel is the guard for senders written in Go, which
 * build a fixed data shape for a fixed email.  Emails named by
 * a Template do not use it: there the data and the email name
 * are authored together, and the load-time RequiredKeys check
 * already verifies the keys line up.
 ******************************************/

// testServerEmailWithModel returns a ServerEmail holding one "test-email" definition for modelName
func testServerEmailWithModel(t *testing.T, modelName string) *ServerEmail {

	t.Helper()

	service := testServerEmail()

	definition := []byte(`{
		emailId: test-email
		model: ` + modelName + `
		to: "{{.To}}"
		subject: "Hello"
	}`)

	require.NoError(t, service.Add(testFilesystem(), definition))
	return &service
}

// TestServerEmail_RequireModel verifies that an email declared for the caller's model is accepted
func TestServerEmail_RequireModel(t *testing.T) {

	service := testServerEmailWithModel(t, "Follower")

	require.NoError(t, service.RequireModel("test-email", "Follower"))
}

// TestServerEmail_RequireModel_Mismatch verifies that an email declared for a different object
// is refused
func TestServerEmail_RequireModel_Mismatch(t *testing.T) {

	service := testServerEmailWithModel(t, "Identity")

	require.ErrorContains(t, service.RequireModel("test-email", "Follower"), "requires a different model object")
}

// TestServerEmail_RequireModel_EmptyModel verifies that a caller must state a model to assert one
func TestServerEmail_RequireModel_EmptyModel(t *testing.T) {

	service := testServerEmailWithModel(t, "Follower")

	require.ErrorContains(t, service.RequireModel("test-email", ""), "Model is required")
}

// TestServerEmail_RequireModel_UnknownEmail verifies that an undefined email fails the assertion
// rather than passing it vacuously
func TestServerEmail_RequireModel_UnknownEmail(t *testing.T) {

	service := testServerEmail()

	require.ErrorContains(t, service.RequireModel("test-email", "Follower"), "Email is not defined")
}

// TestServerEmailAdd_RequiresTo verifies that a definition must name a recipient
func TestServerEmailAdd_RequiresTo(t *testing.T) {

	service := testServerEmail()

	definition := []byte(`{
		emailId: test-email
		model: Stream
		subject: "Hello"
	}`)

	require.ErrorContains(t, service.Add(testFilesystem(), definition), "must include a 'to' address")
	require.Empty(t, service.emails)
}

// TestServerEmailAdd_RejectsReservedHeaderName verifies that a definition cannot set a header that
// decides who receives the message, who it claims to be from, or how the body is framed
func TestServerEmailAdd_RejectsReservedHeaderName(t *testing.T) {

	table := []struct {
		name       string
		headerName string
	}{
		{"recipient: To", "To"},
		{"recipient: Cc", "Cc"},
		{"recipient: Bcc", "Bcc"},
		{"lowercase is canonicalized first", "bcc"},
		{"identity: From", "From"},
		{"identity: Sender", "Sender"},
		{"identity: Return-Path", "Return-Path"},
		{"library-owned: Date", "Date"},
		{"library-owned: MIME-Version", "MIME-Version"},
		{"library-owned: Content-Type", "Content-Type"},
	}

	for _, testCase := range table {
		t.Run(testCase.name, func(t *testing.T) {

			service := testServerEmail()
			definition := testDefinition(`headers: {"` + testCase.headerName + `": "value"}`)

			require.ErrorContains(t, service.Add(testFilesystem(), definition), "header name is reserved")
			require.Empty(t, service.emails)
		})
	}
}

// TestServerEmailAdd_AllowsReplyTo verifies that Reply-To is not a reserved header name
func TestServerEmailAdd_AllowsReplyTo(t *testing.T) {

	service := testServerEmail()

	require.NoError(t, service.Add(testFilesystem(), testDefinition(`headers: {"Reply-To": "{{.ReplyTo}}"}`)))
	require.Contains(t, service.emails, "test-email")
}

// TestServerEmailAdd_DuplicateReplaces verifies that a second definition with the same emailId
// replaces the first rather than failing
func TestServerEmailAdd_DuplicateReplaces(t *testing.T) {

	// A later filesystem location may override an embedded email, so a duplicate is legal

	service := testServerEmail()

	require.NoError(t, service.Add(testFilesystem(), testDefinition("")))
	require.NoError(t, service.Add(testFilesystem(), testDefinition(`emailRole: overridden`)))

	require.Len(t, service.emails, 1)
	require.Equal(t, "overridden", service.emails["test-email"].EmailRole)
}

/******************************************
 * Shipped Definitions
 ******************************************/

// loadEmbeddedEmail loads one of the email definitions that ships in the binary, from disk
func loadEmbeddedEmail(t *testing.T, folder string, emailID string) model.Email {

	t.Helper()

	filesystem := os.DirFS(filepath.Join("..", "_embed", "templates", folder))

	definition, err := fs.ReadFile(filesystem, "email.hjson")
	require.NoError(t, err)

	service := testServerEmail()
	require.NoError(t, service.Add(filesystem, definition))

	email, exists := service.emails[emailID]
	require.True(t, exists, "definition does not declare emailId %q", emailID)

	return email
}

// TestFollowerActivity_ListUnsubscribe verifies that the shipped follower-activity email emits
// List-Unsubscribe from the pre-formatted UnsubscribeWithBrackets key
func TestFollowerActivity_ListUnsubscribe(t *testing.T) {

	// The RFC 2369 bracketing is Follower.UnsubscribeLinkWithBrackets' job, tested in package
	// model; this pins the wiring

	email := loadEmbeddedEmail(t, "email-follower-activity", "follower-activity")

	message := mail.NewMSG()
	data := mapof.Any{"UnsubscribeWithBrackets": "<https://example.com/unsub?id=1>"}

	require.NoError(t, applyHeaders(message, email, data))
	require.NoError(t, message.GetError())

	require.Contains(t, message.GetMessage(), "List-Unsubscribe: <https://example.com/unsub?id=1>")
}

// TestFollowerActivity_OneClickUnsubscribe verifies the RFC 8058 header that turns a
// List-Unsubscribe link into a one-click button in Gmail and Yahoo
func TestFollowerActivity_OneClickUnsubscribe(t *testing.T) {

	// The value is a fixed literal: the provider POSTs to the List-Unsubscribe address.  The
	// button appears only when BOTH headers are present, so a typo here fails invisibly

	email := loadEmbeddedEmail(t, "email-follower-activity", "follower-activity")

	message := mail.NewMSG()
	data := mapof.Any{"UnsubscribeWithBrackets": "<https://example.com/unsub?id=1>"}

	require.NoError(t, applyHeaders(message, email, data))
	require.NoError(t, message.GetError())

	require.Contains(t, message.GetMessage(), "List-Unsubscribe-Post: List-Unsubscribe=One-Click")
}

// TestFollowerActivity_OneClickRequiresLink verifies that List-Unsubscribe-Post always travels
// with a List-Unsubscribe link
func TestFollowerActivity_OneClickRequiresLink(t *testing.T) {

	// Without the link, one-click invites a POST to a URL the recipient never received.  The
	// pairing is structural: Follower.UnsubscribeLinkWithBrackets always returns a link

	email := loadEmbeddedEmail(t, "email-follower-activity", "follower-activity")

	message := mail.NewMSG()
	data := mapof.Any{"UnsubscribeWithBrackets": model.NewFollower().UnsubscribeLinkWithBrackets("https://example.com")}

	require.NoError(t, applyHeaders(message, email, data))
	require.NoError(t, message.GetError())

	// The List-Unsubscribe value is long enough that go-simple-mail folds it onto its own line,
	// so the URL is asserted separately from the header name that carries it
	require.Contains(t, message.GetMessage(), "List-Unsubscribe: ")
	require.Contains(t, message.GetMessage(), "<https://example.com/")
	require.Contains(t, message.GetMessage(), "List-Unsubscribe-Post: List-Unsubscribe=One-Click")
}

// TestEmbeddedEmails_Load verifies that every email definition shipped in the binary loads
// through Add(), the same way the Template service loads it at startup
func TestEmbeddedEmails_Load(t *testing.T) {

	// A rule in Add() that rejects one of Emissary's own definitions fails here, not at a
	// customer's next boot.  Sibling of TestEmbeddedTemplates_HTMLParses

	const root = "../_embed/templates"

	entries, err := os.ReadDir(root)
	require.NoError(t, err)

	service := testServerEmail()
	loaded := make([]string, 0)

	for _, entry := range entries {

		if !entry.IsDir() {
			continue
		}

		filesystem := os.DirFS(filepath.Join(root, entry.Name()))
		definitionType, definition := findDefinition(filesystem)

		if definitionType != DefinitionEmail {
			continue
		}

		require.NoError(t, service.Add(filesystem, definition), "email definition %q does not load", entry.Name())
		loaded = append(loaded, entry.Name())
	}

	// A silent zero would make this test pass forever if the directory ever moved
	require.NotEmpty(t, loaded, "no email definitions found under "+root)

	// Every definition must claim its own emailId, or one silently replaces another
	require.Len(t, service.emails, len(loaded), "two shipped definitions share an emailId")
}

/******************************************
 * Load-Time Queries
 ******************************************/

// TestServerEmailExists verifies the lookup that load-time validation uses to confirm a
// send-email step names a definition that was actually loaded
func TestServerEmailExists(t *testing.T) {

	service := testServerEmail()

	require.False(t, service.Exists("test-email"))
	require.NoError(t, service.Add(testFilesystem(), testDefinition("")))
	require.True(t, service.Exists("test-email"))
}

// TestServerEmailRequiredKeys verifies that the keys an email's "to", "subject", and "headers"
// templates interpolate are reported
func TestServerEmailRequiredKeys(t *testing.T) {

	service := testServerEmail()
	definition := []byte(`{
		emailId: test-email
		model: Stream
		to: "{{.Recipient}}"
		subject: "Hello {{.SubjectOnly}}"
		headers: {"Reply-To": "{{.ReplyEmail}}"}
	}`)

	require.NoError(t, service.Add(testFilesystem(), definition))

	// Subject IS included and the body is not; RequiredKeys explains why
	require.Equal(t, []string{"Recipient", "ReplyEmail", "SubjectOnly"}, []string(service.RequiredKeys("test-email")))
}

// TestServerEmailRequiredKeys_ExcludesProvided verifies that the Domain_* values DomainEmail.Send
// supplies for every email are not reported
func TestServerEmailRequiredKeys_ExcludesProvided(t *testing.T) {

	// Callers never pass these keys (D20 in CONTACT-FORM)

	service := testServerEmail()
	definition := []byte(`{
		emailId: test-email
		model: Stream
		to: "{{.Recipient}}"
		subject: "Hello"
		headers: {"X-Domain": "{{.Domain_Name}} {{.Domain_URL}}"}
	}`)

	require.NoError(t, service.Add(testFilesystem(), definition))
	require.Equal(t, []string{"Recipient"}, []string(service.RequiredKeys("test-email")))
}

// TestServerEmailRequiredKeys_GuardedKeysCount verifies that a key inside an {{if}} is still
// reported
func TestServerEmailRequiredKeys_GuardedKeysCount(t *testing.T) {

	// Under missingkey=error an absent key fails the guard itself, so it is required

	service := testServerEmail()
	definition := []byte(`{
		emailId: test-email
		model: Stream
		to: "{{.Recipient}}"
		subject: "Hello"
		headers: {"List-Unsubscribe": "{{if .Unsubscribe}}<{{.Unsubscribe}}>{{end}}"}
	}`)

	require.NoError(t, service.Add(testFilesystem(), definition))
	require.Equal(t, []string{"Recipient", "Unsubscribe"}, []string(service.RequiredKeys("test-email")))
}

// TestServerEmailRequiredKeys_ShippedDefinition pins the keys that the shipped follower-activity
// email requires
func TestServerEmailRequiredKeys_ShippedDefinition(t *testing.T) {

	service := testServerEmail()
	filesystem := os.DirFS(filepath.Join("..", "_embed", "templates", "email-follower-activity"))

	definition, err := fs.ReadFile(filesystem, "email.hjson")
	require.NoError(t, err)
	require.NoError(t, service.Add(filesystem, definition))

	// "Name" comes from the subject line, "New Activity From {{.Name}}"
	require.Equal(t, []string{"Email", "Name", "UnsubscribeWithBrackets"}, []string(service.RequiredKeys("follower-activity")))
}

/******************************************
 * Contact Form Email
 *
 * The contact form is the first email whose data comes from an
 * anonymous visitor rather than from Emissary itself, so these
 * tests pin the two properties that keep it safe: nothing the
 * visitor writes is trusted as markup, and no template here
 * references a key that nobody supplies.
 ******************************************/

// contactFormEmail loads the shipped contact-form definition the same way the server does
func contactFormEmail(t *testing.T) (*ServerEmail, model.Email) {

	t.Helper()

	service := testServerEmail()
	filesystem := os.DirFS("../_embed/templates/email-contact-form")

	definitionType, definition := findDefinition(filesystem)
	require.Equal(t, DefinitionEmail, definitionType)
	require.NoError(t, service.Add(filesystem, definition))

	email, exists := service.emails["contact-form"]
	require.True(t, exists, "the shipped definition must declare emailId 'contact-form'")

	return &service, email
}

// contactFormContract is every key the contact-form templates may reference: six message keys and
// twelve Client_* keys from the send-email step, plus four from DomainEmail.Send
var contactFormContract = []string{
	"To", "Subject", "ReplyEmail", "Name", "Message", "HeaderMessage",
	"Client_IP", "Client_Description", "Client_Referer", "Client_UserAgent",
	"Client_Brands", "Client_Platform", "Client_Mobile", "Client_AcceptLanguage",
	"Client_Accept", "Client_AcceptEncoding", "Client_DoNotTrack", "Client_PrivacyControl",
	"Domain_Owner", "Domain_URL", "Domain_Name", "Domain_Icon",
}

// TestContactFormEmail_RequiredKeys pins the keys that the Stream template's send-email step
// must supply to the contact-form email
func TestContactFormEmail_RequiredKeys(t *testing.T) {

	// Pinning the set makes a mismatch fail when Templates load, rather than the first time a
	// visitor submits the form

	service, _ := contactFormEmail(t)

	require.Equal(t, []string{"ReplyEmail", "Subject", "To"}, []string(service.RequiredKeys("contact-form")))
}

// TestContactFormEmail_KeysAreInTheContract verifies that every key the contact-form subject and
// body reference is in contactFormContract
func TestContactFormEmail_KeysAreInTheContract(t *testing.T) {

	// RequiredKeys skips the body, where a missing key renders blank with no error.  Extending it
	// would break email-follower-activity, whose body references .URL and .Actor that nothing
	// supplies, so this covers only the email whose contract is fully known.

	_, email := contactFormEmail(t)

	found := make(mapof.Bool)
	collectTreeFieldNames(email.Subject, found)

	require.NotNil(t, email.Body.Tree, "the body must be parsed before its tree can be walked")
	collectFieldNames(email.Body.Tree.Root, found)

	require.NotEmpty(t, found, "a walk that finds nothing would pass forever")

	for key := range found {
		require.Contains(t, contactFormContract, key,
			"body.html or email.hjson references %q, which no caller supplies -- it will render empty, with no error anywhere", key)
	}
}

// TestContactFormEmail_EscapesVisitorInput verifies that everything an anonymous visitor writes
// is escaped rather than rendered
func TestContactFormEmail_EscapesVisitorInput(t *testing.T) {

	// This holds while visitor values stay plain interpolations.  Piping one through markdown,
	// htmlMinimal, highlight, or any helper returning template.HTML marks it safe, and hands a
	// stranger script execution in the recipient's mail client.

	_, email := contactFormEmail(t)

	data := mapof.Any{
		"Name":          `<script>alert("name")</script>`,
		"ReplyEmail":    `"onmouseover="alert(1)`,
		"Message":       `<script>alert("message")</script>` + "\n<b>bold</b>",
		"HeaderMessage": "",
		"Domain_Icon":   "",
		"Domain_Name":   "Example",

		// The Client_* values are headers, so they are attacker-controlled in the same
		// way the three fields above are -- a client chooses every byte of its own User-Agent.
		"Client_IP":         "203.0.113.42",
		"Client_UserAgent":  `<script>alert("ua")</script>`,
		"Client_Referer":    `javascript:alert(1)`,
		"Client_Platform":   `"onmouseover="alert(2)`,
		"Client_DoNotTrack": `<img src=x onerror=alert(3)>`,
	}

	var buffer strings.Builder
	require.NoError(t, email.Body.Execute(&buffer, data))

	result := buffer.String()

	require.NotContains(t, result, "<script>", "visitor input must never reach the recipient as markup")
	require.NotContains(t, result, "<b>bold</b>")
	require.Contains(t, result, "&lt;script&gt;", "the script tag must survive as escaped text")
	require.NotContains(t, result, `"onmouseover="alert(1)`, "an address must not break out of its attribute")
	require.NotContains(t, result, `"onmouseover="alert(2)`, "a header value must not break out of its cell")
	require.NotContains(t, result, "<img src=x", "a header value must never reach the recipient as markup")

	// RULE: the Referer is rendered as TEXT, never as a link.  If it ever becomes an href,
	// html/template rewrites this to "#ZgotmplZ" -- which hides from the reader the one thing
	// the row exists to show them.  Asserting the raw text survives pins the display choice.
	require.Contains(t, result, "javascript:alert(1)", "the Referer must be shown verbatim, as text")
	require.NotContains(t, result, "ZgotmplZ", "no Client_* value belongs in a URL context")

	// The IP is the one value that IS a link, which is safe only because every realclientip
	// strategy returns "" for anything it cannot parse as an address.
	require.Contains(t, result, `href="https://ipinfo.io/203.0.113.42"`, "the IP must link to a lookup the recipient can follow")
}

// TestContactFormEmail_RendersAuthorMarkdown verifies that the page author's header message is
// converted, since it is the one value here that is meant to carry formatting
func TestContactFormEmail_RendersAuthorMarkdown(t *testing.T) {

	_, email := contactFormEmail(t)

	data := mapof.Any{
		"Name":          "Sarah",
		"ReplyEmail":    "sarah@example.com",
		"Message":       "Hello",
		"HeaderMessage": "Please allow **two days** for a reply.",
		"Domain_Icon":   "",
		"Domain_Name":   "Example",
	}

	var buffer strings.Builder
	require.NoError(t, email.Body.Execute(&buffer, data))

	require.Contains(t, buffer.String(), "<strong>two days</strong>", "the author's Markdown must be converted")
}
