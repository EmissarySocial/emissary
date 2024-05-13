package build

// ActionMethod enumerates the HTTP methods that can be performed on Actions
type ActionMethod uint8

// ActionMethodGet signifies a GET operation on an action
const ActionMethodGet ActionMethod = 0

// ActionMethodPost signifies a POST operation on an action
const ActionMethodPost ActionMethod = 1
