package derpmongo

import (
	"fmt"

	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
)

// encodableError is a derp.Error rebuilt from parts that BSON can always encode.  Its field
// names match derp.Error's default BSON names, so triage reads both shapes the same way.
type encodableError struct {
	Code         int    `bson:"code"`
	Location     string `bson:"location"`
	Message      string `bson:"message"`
	URL          string `bson:"url"`
	Details      []any  `bson:"details"`
	TimeStamp    int64  `bson:"timestamp"`
	WrappedValue error  `bson:"wrappedvalue"`
}

// Error implements the error interface
func (err encodableError) Error() string {
	return err.Message
}

// makeEncodable rebuilds an error chain, replacing any value that BSON cannot encode
func makeEncodable(err error) error {

	if err == nil {
		return nil
	}

	// A derp.Error is rebuilt field by field, so one bad detail costs only that detail
	switch typed := err.(type) {
	case derp.Error:
		return encodableDerpError(typed)
	case *derp.Error:
		if typed != nil {
			return encodableDerpError(*typed)
		}
	}

	// Any other error is kept whole when it encodes, and reduced to its message when it does not
	if isEncodable(err) {
		return err
	}

	return encodableError{Message: err.Error()}
}

// encodableDerpError rebuilds one derp.Error, and the chain beneath it
func encodableDerpError(err derp.Error) encodableError {
	return encodableError{
		Code:         err.Code,
		Location:     err.Location,
		Message:      err.Message,
		URL:          err.URL,
		Details:      encodableDetails(err.Details),
		TimeStamp:    err.TimeStamp,
		WrappedValue: makeEncodable(err.WrappedValue),
	}
}

// encodableDetails replaces each detail that BSON cannot encode with the name of its type
func encodableDetails(details []any) []any {

	result := make([]any, 0, len(details))

	for _, detail := range details {

		if isEncodable(detail) {
			result = append(result, detail)
			continue
		}

		// RULE: Store only the type, never fmt's dump of the value, which prints every field
		result = append(result, fmt.Sprintf("unencodable value of type %T", detail))
	}

	return result
}

// isEncodable reports whether BSON can encode the value
func isEncodable(value any) bool {
	_, err := bson.Marshal(bson.M{"value": value})
	return err == nil
}
