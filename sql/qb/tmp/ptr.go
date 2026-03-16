// Package tmp is a temporary holding package for pointer helper functions
// pending future refactoring into a shared utility package.
//
// NOTE: This package is a temporary home for helpers relocated from
// devops-api-service (tbutil). It should be refactored or merged into a
// proper utility package in a future cleanup effort.
package tmp

// ToStringPtr returns a pointer to the supplied string value.
func ToStringPtr(val string) *string {
	return &val
}

// ToInt64Ptr returns a pointer to the supplied int64 value.
func ToInt64Ptr(val int64) *int64 {
	return &val
}
