package mongoUtils

import (
	"fmt"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"reflect"
)

var typeEncoderDecoderMap = map[reflect.Type]func() (bsoncodec.ValueEncoderFunc, bsoncodec.ValueDecoderFunc){
	UUIDType: func() (bsoncodec.ValueEncoderFunc, bsoncodec.ValueDecoderFunc) {
		return UuidEncodeValue, UuidDecodeValue
	},
}

func NewRegistry(types ...reflect.Type) *bsoncodec.Registry {
	r := bson.NewRegistry()

	for _, t := range types {
		encoder, decoder := typeEncoderDecoderMap[t]()
		r.RegisterTypeEncoder(t, encoder)
		r.RegisterTypeDecoder(t, decoder)
	}

	return r
}

var (
	UUIDType    = reflect.TypeOf(uuid.UUID{})
	uuidSubtype = byte(0x04)
)

func UuidEncodeValue(_ bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != UUIDType {
		return bsoncodec.ValueEncoderError{Name: "uuidEncodeValue", Types: []reflect.Type{UUIDType}, Received: val}
	}
	b := val.Interface().(uuid.UUID)
	return vw.WriteBinaryWithSubtype(b[:], uuidSubtype)
}

func UuidDecodeValue(_ bsoncodec.DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != UUIDType {
		return bsoncodec.ValueDecoderError{Name: "uuidDecodeValue", Types: []reflect.Type{UUIDType}, Received: val}
	}

	var data []byte
	var subtype byte
	var err error
	switch vrType := vr.Type(); vrType {
	case bson.TypeBinary:
		data, subtype, err = vr.ReadBinary()
		if subtype != uuidSubtype {
			return fmt.Errorf("unsupported binary subtype %v for UUID", subtype)
		}
	case bson.TypeNull:
		err = vr.ReadNull()
	case bson.TypeUndefined:
		err = vr.ReadUndefined()
	default:
		return fmt.Errorf("cannot decode %v into a UUID", vrType)
	}

	if err != nil {
		return err
	}
	uuid2, err := uuid.FromBytes(data)
	if err != nil {
		return err
	}
	val.Set(reflect.ValueOf(uuid2))
	return nil
}
