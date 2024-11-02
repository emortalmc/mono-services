package registrytypes

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"reflect"
	"time"
)

var (
	DurationType    = reflect.TypeOf(time.Duration(0))
	NanosInMillisecond = time.Millisecond.Nanoseconds()
)

func DurationEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != DurationType {
		return bsoncodec.ValueEncoderError{Name: "durationEncodeValue", Types: []reflect.Type{DurationType}, Received: val}
	}
	b := val.Interface().(time.Duration)

	return vw.WriteInt64(b.Milliseconds())
}

func DurationDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != DurationType {
		return bsoncodec.ValueDecoderError{Name: "durationDecodeValue", Types: []reflect.Type{DurationType}, Received: val}
	}

	var data int64
	var err error
	switch vrType := vr.Type(); vrType {
	case bson.TypeInt64:
		data, err = vr.ReadInt64()
	default:
		return bsoncodec.ValueDecoderError{Name: "durationDecodeValue", Types: []reflect.Type{DurationType}, Received: val}
	}

	if err != nil {
		return err
	}

	val.Set(reflect.ValueOf(time.Duration(data * NanosInMillisecond)))
	return nil
}
