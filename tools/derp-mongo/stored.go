package derpmongo

import (
	"fmt"
	"strings"

	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
)

// dupKeyMarker begins the part of a MongoDB duplicate-key message that quotes the key's value
const dupKeyMarker = "dup key:"

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

// StoredError returns the error chain exactly as Report stores it: encodable, and with every
// duplicated key cut away.  The original chain is never modified.
func StoredError(err error) error {

	// RULE: derp.IsNil catches a typed nil, which derp.Wrap stores as a WrappedValue and whose
	// Error() would panic below.  The second check is the one nilaway can see.
	if derp.IsNil(err) || err == nil {
		return nil
	}

	// A derp.Error is rebuilt field by field, so one bad detail costs only that detail
	switch typed := err.(type) {
	case derp.Error:
		return storedDerpError(typed)
	case *derp.Error:
		if typed != nil {
			return storedDerpError(*typed)
		}
	}

	// RULE: A foreign layer is kept whole only when it encodes and quotes no key.  Anything
	// else is reduced to its message, which is all that triage reads from it.
	if isEncodable(err) && !strings.Contains(err.Error(), dupKeyMarker) {
		return err
	}

	return encodableError{Message: cutDupKey(err.Error())}
}

// storedDerpError rebuilds one derp.Error, and the chain beneath it
func storedDerpError(err derp.Error) encodableError {
	return encodableError{
		Code:         err.Code,
		Location:     err.Location,
		Message:      cutDupKey(err.Message),
		URL:          err.URL,
		Details:      storedDetails(err.Details),
		TimeStamp:    err.TimeStamp,
		WrappedValue: StoredError(err.WrappedValue),
	}
}

// storedDetails cuts every string detail, and replaces each unencodable one with its type name
func storedDetails(details []any) []any {

	result := make([]any, 0, len(details))

	for _, detail := range details {

		// derp.Wrap appends a foreign inner error's message as a string detail
		if text, isString := detail.(string); isString {
			result = append(result, cutDupKey(text))
			continue
		}

		if isEncodable(detail) {
			result = append(result, detail)
			continue
		}

		// RULE: Store only the type, never fmt's dump of the value, which prints every field
		result = append(result, fmt.Sprintf("unencodable value of type %T", detail))
	}

	return result
}

// cutDupKey removes the duplicated key that a MongoDB E11000 message quotes, which may be
// personal data or a secret
func cutDupKey(message string) string {

	if index := strings.Index(message, dupKeyMarker); index >= 0 {
		return strings.TrimSpace(message[:index])
	}

	return message
}

// isEncodable reports whether BSON can encode the value
func isEncodable(value any) bool {
	_, err := bson.Marshal(bson.M{"value": value})
	return err == nil
}
