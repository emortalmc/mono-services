package registrytypes

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
)

var CodecRegistry = createCodecRegistry()

func createCodecRegistry() *bsoncodec.Registry {
	r := bson.NewRegistry()

	r.RegisterTypeEncoder(UUIDType, bsoncodec.ValueEncoderFunc(UuidEncodeValue))
	r.RegisterTypeDecoder(UUIDType, bsoncodec.ValueDecoderFunc(UuidDecodeValue))

	return r
}
