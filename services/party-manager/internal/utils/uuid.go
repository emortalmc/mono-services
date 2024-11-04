package utils

import "github.com/google/uuid"

func RandomUuidSlice(len int) []uuid.UUID {
	uuids := make([]uuid.UUID, len)
	for i := 0; i < len; i++ {
		uuids[i] = uuid.New()
	}

	return uuids
}
