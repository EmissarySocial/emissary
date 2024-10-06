package queue

// Marshaller is an interface that wraps the Marshal and Unmarshal method,
// which is used to retrieve Tasks from the Storage system.
type Marshaller interface {

	// Marshal exports a Task into a map[string]any object,
	// which can be saved to the Storage provider
	Marshal(task Task) (map[string]any, bool)

	// Unmarshal imports a Task from a map[string]any object,
	// which has been retrieved from the Storage provider
	Unmarshal(journal *Journal) bool
}
