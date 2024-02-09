package jsontemplate

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func ExampleTemplate_Execute() {

	template, _ := New(`{"template": "Hello, {{.name}}!"}`)

	data := map[string]any{"name": "World"}
	result := make(map[string]any)

	if err := template.Execute(&result, data); err != nil {
		fmt.Println("Error: ", err)
	}

	fmt.Println(result["template"]) // Output: Hello, World!
}

func TestEscapedCharacters(t *testing.T) {

	template, _ := New(`{"message": "{{.message}}"}`)

	values := []string{"With an A'postrophe", `With "Double Quotes" As Well`, "With <HTML> Tags"}

	for _, token := range values {

		t.Log(token)
		data := map[string]any{"message": token}
		result := make(map[string]any)

		err := template.Execute(&result, data)
		require.Nil(t, err)
		require.Equal(t, token, result["message"])
	}
}

func TestCrazyArrays(t *testing.T) {

	template, _ := New(`[{{range .values}}"{{.}}",{{end}}]`)

	data := map[string]any{
		"values": []string{"one", "two", "three"},
	}
	result := make([]string, 0, 3)

	err := template.Execute(&result, data)
	require.Nil(t, err)
	require.Equal(t, result[0], "one")
	require.Equal(t, result[1], "two")
	require.Equal(t, result[2], "three")
}
