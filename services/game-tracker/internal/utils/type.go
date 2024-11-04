package utils

import "reflect"

func IsSameType(a, b interface{}) bool {
	aType := reflect.TypeOf(a)
	bType := reflect.TypeOf(b)

	// Check if both types are exactly the same
	if aType == bType {
		return true
	}

	// Check if one of the types is a pointer and the other is the underlying element type
	if aType.Kind() == reflect.Ptr && aType.Elem() == bType {
		return true
	}

	if bType.Kind() == reflect.Ptr && bType.Elem() == aType {
		return true
	}

	return false
}
