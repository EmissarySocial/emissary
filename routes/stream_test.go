package routes

import (
	"testing"

	"github.com/benpate/remote"
	"github.com/stretchr/testify/assert"
)

func TestStream(t *testing.T) {

	go startTestServer()

	{
		result := ""
		err := ""
		txn := remote.Get("http://localhost:8080/omg-it-works").
			Response(&result, &err)

		assert.Nil(t, txn.Send())

		t.Log(result)
		t.Log(err)
		t.Fail()
	}
}
