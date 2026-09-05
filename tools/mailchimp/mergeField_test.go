package mailchimp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGetMergeFields reads a Mailchimp response into typed values
func TestGetMergeFields(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {

		require.Equal(t, "/lists/abc123/merge-fields", request.URL.Path)

		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"merge_fields":[
			{"merge_id":1,"tag":"FNAME","name":"First Name","type":"text"},
			{"merge_id":9,"tag":"EMISSARYID","name":"Emissary ID","type":"text"}
		]}`))
	}))

	defer server.Close()

	mergeFields, err := testClient(server.URL).GetMergeFields("abc123")

	require.NoError(t, err)
	require.Len(t, mergeFields, 2)
	require.Equal(t, "EMISSARYID", mergeFields[1].Tag)
	require.Equal(t, 9, mergeFields[1].MergeID)
}

// TestCreateMergeField_IsNeitherRequiredNorPublic pins the two flags that decide whether
// this field is harmless
func TestCreateMergeField_IsNeitherRequiredNorPublic(t *testing.T) {

	// A `required` merge field breaks every other way members enter this audience --
	// including the User's own Mailchimp signup forms, which cannot supply an Emissary ID.
	// A `public` one puts that ID in front of their subscribers.  Neither flag is left to
	// a Mailchimp default.

	var body map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {

		require.Equal(t, http.MethodPost, request.Method)
		require.Equal(t, "/lists/abc123/merge-fields", request.URL.Path)

		raw, _ := io.ReadAll(request.Body)
		require.NoError(t, json.Unmarshal(raw, &body))

		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"merge_id":9,"tag":"EMISSARYID","name":"Emissary ID","type":"text"}`))
	}))

	defer server.Close()

	mergeField, err := testClient(server.URL).CreateMergeField("abc123", "EMISSARYID", "Emissary ID")

	require.NoError(t, err)
	require.Equal(t, 9, mergeField.MergeID)

	require.Equal(t, "EMISSARYID", body["tag"])
	require.Equal(t, "text", body["type"])
	require.Equal(t, false, body["required"])
	require.Equal(t, false, body["public"])
}

// TestCreateMergeField_ErrorsAreActionable confirms a full audience reads as an instruction
func TestCreateMergeField_ErrorsAreActionable(t *testing.T) {

	// An audience at its 30-field cap refuses the create, and that is a legitimate answer
	// about the User's own account rather than a fault in this server.

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusBadRequest)
	}))

	defer server.Close()

	_, err := testClient(server.URL).CreateMergeField("abc123", "EMISSARYID", "Emissary ID")
	require.Error(t, err)
}
