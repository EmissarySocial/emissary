package build

import (
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/EmissarySocial/emissary/tools/formdata"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

// StepReadForm is a Step that reads named form fields into the Builder's temporary data scope
type StepReadForm struct {
	Schema schema.Schema
}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepReadForm) Get(_ Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post copies the approved form fields into the Builder's temporary data scope
func (step StepReadForm) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepReadForm.Post"

	// RULE: read the request BODY only.  formdata.Parse merges the URL query string into
	// its result, and a repeated name yields BOTH values joined below -- so a crafted link
	// could append attacker-controlled text to a field the visitor believes is theirs, and
	// the message would go out carrying it.
	transaction, err := formdata.ParseBody(builder.request())

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error parsing form data", derp.WithBadRequest()))
	}

	// RULE: the schema names every field this step accepts, so a field the template did not
	// declare is never read.  Visitor input reaches the Builder's temporary scope and NEVER
	// the object being built, which for a Stream is the page record itself.
	values := mapof.NewAny()

	for name := range step.Schema.AllProperties() {

		value := strings.Join(transaction[name], ",")

		// RULE: reject an over-long value instead of storing a shortened one, matching
		// "edit-content".  schema.Set TRUNCATES to maxLength and reports success, and Validate
		// then passes the shortened value -- so an over-long address is stored as a DIFFERENT,
		// still-valid address, and an over-long message is delivered half-sent.  Length is
		// counted in runes, because that is the unit maxLength uses everywhere in this codebase
		// (and the unit schema.Set truncates by); counting bytes would reject text that fits.
		if element, ok := step.Schema.GetStringElement(name); ok {
			if (element.MaxLength > 0) && (utf8.RuneCountInString(value) > element.MaxLength) {
				err := derp.BadRequest(location, "Value is too long", name, "maximum: "+strconv.Itoa(element.MaxLength))
				return Halt().WithError(err)
			}
		}

		// Set through the schema so that each value is validated and coerced
		if err := step.Schema.Set(&values, name, value); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Invalid value", name, derp.WithBadRequest()))
		}
	}

	// RULE: every declared field must survive validation, including the ones the visitor
	// left out, so a missing "required" field fails here rather than rendering as empty
	if _, err := step.Schema.Validate(values); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Form data is not valid", derp.WithBadRequest()))
	}

	// Publish into the temporary scope, where later steps read them via GetString
	for name, value := range values {
		builder.setString(name, convert.String(value))
	}

	// Form and function
	return nil
}
