package ashash

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClient_WithoutHash_Success(t *testing.T) {

	client := New(testInnerClient{})

	result, err := client.Load("http://example.com/without-hash")
	require.Nil(t, err)
	require.Equal(t, "Without Hash", result.Name())
	require.Equal(t, "Ain't nobody got no hash", result.Summary())
}

func TestClient_WithoutHash_Fail(t *testing.T) {

	client := New(testInnerClient{})

	result, err := client.Load("http://example.com/without-hash#but-hash-anyway")
	require.Error(t, err)
	require.True(t, result.IsNil())
	require.Equal(t, "", result.Name())
	require.Equal(t, "", result.Summary())
}

func TestClient_WithHash_Success1(t *testing.T) {

	client := New(testInnerClient{})

	result, err := client.Load("http://example.com/with-hash")
	require.Nil(t, err)
	require.Equal(t, "With Hash", result.Name())
	require.Equal(t, "It's my hash and I can cry if I want to", result.Summary())
}

func TestClient_WithHash_Success2(t *testing.T) {

	client := New(testInnerClient{})

	result, err := client.Load("http://example.com/with-hash#hash")
	require.Nil(t, err)
	require.Equal(t, "Here's the Hash", result.Name())
	require.Equal(t, "Done somebody gots a hash, now.", result.Summary())
}

func TestClient_WithHash_Fail(t *testing.T) {

	client := New(testInnerClient{})

	result, err := client.Load("http://example.com/with-hash#bad-hash")
	require.Error(t, err)
	require.True(t, result.IsNil())
	require.Equal(t, "", result.Name())
	require.Equal(t, "", result.Summary())
}
