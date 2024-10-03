package queue

// Unmarshaller is an interface that wraps the Unmarshal method,
// which is used to retrieve Tasks from the Storage system.
type Unmarshaller interface {
	Unmarshal(journal *Journal) error
}
