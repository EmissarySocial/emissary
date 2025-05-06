package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
)

// StepWithProduct is a Step that can update the data.DataMap custom data stored in a Stream
type StepWithProduct struct {
	SubSteps []step.Step
}

func (step StepWithProduct) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodGet)
}

// Post updates the stream with approved data from the request body.
func (step StepWithProduct) Post(builder Builder, buffer io.Writer) PipelineBehavior {
	return step.execute(builder, buffer, ActionMethodPost)
}

func (step StepWithProduct) execute(builder Builder, buffer io.Writer, actionMethod ActionMethod) PipelineBehavior {

	const location = "build.StepWithProduct.doStep"

	if !builder.IsAuthenticated() {
		return Halt().WithError(derp.UnauthorizedError(location, "Anonymous user is not authorized to perform this action"))
	}

	// Try to find the Template for this builder.
	// This *should* work for all builders that use CommonWithTemplate
	template, exists := getTemplate(builder)

	if !exists {
		return Halt().WithError(derp.InternalError(location, "This step cannot be used in this Renderer."))
	}

	// Collect required services and values
	factory := builder.factory()
	productService := factory.Product()
	token := builder.QueryParam("productId")
	product := model.NewProduct()
	product.UserID = builder.AuthenticatedID()

	if (token != "") && (token != "new") {
		if err := productService.LoadByUserAndToken(builder.AuthenticatedID(), token, &product); err != nil {
			if actionMethod == ActionMethodGet {
				return Halt().WithError(derp.Wrap(err, location, "Unable to load Product", token))
			}
			// Fall through for POSTS..  we're just creating a new product.
		}
	}

	// Create a new builder tied to the Product record
	subBuilder, err := NewModel(factory, builder.request(), builder.response(), template, &product, builder.actionID())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to create sub-builder"))
	}

	// Execute the POST build pipeline on the child
	reesult := Pipeline(step.SubSteps).Execute(factory, subBuilder, buffer, actionMethod)
	reesult.Error = derp.Wrap(reesult.Error, location, "Error executing steps for child")

	return UseResult(reesult)
}
