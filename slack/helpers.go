package slack

// toStringPtr returns a pointer to a copy of val.
// It is used internally to convert string literals to the *string fields
// expected by Slack API request and response types.
func toStringPtr(val string) *string {
	return &val
}

// toInt64Ptr returns a pointer to a copy of val.
// It is used internally to convert int64 literals to the *int64 fields
// expected by Slack API request types.
func toInt64Ptr(val int64) *int64 {
	return &val
}
