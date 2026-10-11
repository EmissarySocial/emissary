package service

import (
	"os"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	modelStep "github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/rosetta/schema"
	"github.com/hjson/hjson-go/v4"
	"github.com/stretchr/testify/require"
)

// TestActorButtonBlockUpdate_PostsRuleFields confirms that every field the block popup
// posts is one the Rule schema accepts, and that the popup's textarea posts it (BUG-235)
func TestActorButtonBlockUpdate_PostsRuleFields(t *testing.T) {

	for _, templateID := range []string{"user-inbox", "user-settings"} {

		t.Run(templateID, func(t *testing.T) {

			definition, err := os.ReadFile("../_embed/templates/" + templateID + "/template.hjson")
			require.NoError(t, err)

			template := model.NewTemplate(templateID, nil)
			require.NoError(t, hjson.Unmarshal(definition, &template))

			action, exists := template.Actions["actor-button-block-update"]
			require.True(t, exists)

			popup, err := os.ReadFile("../_embed/templates/" + templateID + "/actor-button-block.html")
			require.NoError(t, err)

			ruleSchema := schema.New(model.RuleSchema())
			paths := 0

			for _, step := range allSteps(action.Steps) {

				setData, ok := step.(modelStep.SetData)

				if !ok {
					continue
				}

				for _, path := range setData.FromForm {
					rule := model.NewRule()
					require.NoError(t, ruleSchema.Set(&rule, path, "Spams every thread"), "the Rule schema must accept %q", path)
					require.Contains(t, string(popup), `name="`+path+`"`, "the popup must post %q", path)
					paths++
				}
			}

			require.Positive(t, paths, "the action must set at least one field from the form")
		})
	}
}
