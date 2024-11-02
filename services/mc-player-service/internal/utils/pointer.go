package utils

func PointerOf[T any](v T) *T {
	return &v
}
