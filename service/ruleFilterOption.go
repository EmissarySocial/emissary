package service

// RuleFilterOption defines a functional option that modifies the
// behavior of a RuleFilter object
type RuleFilterOption func(*RuleFilter)

// IgnoreBlocks returns a RuleFilterOption that prevents `Block` -type
// rules from being executed
func IgnoreBlocks() RuleFilterOption {
	return func(filter *RuleFilter) {
		filter.allowBlocks = false
	}
}

// IgnoreMutes returns a RuleFilterOption that prevents `Mute` -type
// rules from being executed
func IgnoreMutes() RuleFilterOption {
	return func(filter *RuleFilter) {
		filter.allowMutes = false
	}
}

// IgnoreLabels returns a RuleFilterOption that prevents `Label` -type
// rules from being executed
func IgnoreLabels() RuleFilterOption {
	return func(filter *RuleFilter) {
		filter.allowLabels = false
	}
}

// WithLabelsOnly returns a RuleFilterOption that allows ONLY `Label` -type
// rules, and blocks all others.
func WithLabelsOnly() RuleFilterOption {
	return func(filter *RuleFilter) {
		filter.allowBlocks = false
		filter.allowMutes = false
		filter.allowLabels = true
	}
}
