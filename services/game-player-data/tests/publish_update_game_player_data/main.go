package main

import (
	"context"
	kafka2 "github.com/emortalmc/mono-services/services/game-player-data/internal/kafka"
	pbmsg "github.com/emortalmc/proto-specs/gen/go/message/gameplayerdata"
	pbmodel "github.com/emortalmc/proto-specs/gen/go/model/gameplayerdata"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"time"
)

const pIdStr = "8d36737e-1c0a-4a71-87de-9906f577845e"

func main() {
	w := &kafka.Writer{
		Addr:      kafka.TCP("localhost:9092"),
		Topic:     kafka2.GamePlayerDataTopic,
		BatchSize: 1,
		Async:     false,
	}

	data := &pbmodel.V1BlockSumoPlayerData{
		BlockSlot:  12,
		ShearsSlot: 13,
	}

	dataAny, err := anypb.New(data)
	if err != nil {
		panic(err)
	}

	msg := &pbmsg.UpdateGamePlayerDataMessage{
		PlayerId: pIdStr,
		GameMode: pbmodel.GameDataGameMode_BLOCK_SUMO,
		Data:     dataAny,
		DataMask: &fieldmaskpb.FieldMask{Paths: []string{"block_slot", "shears_slot"}},
	}

	if err := w.WriteMessages(context.Background(), createKafkaMessage(msg)); err != nil {
		panic(err)
	}
}

func createKafkaMessage(pb proto.Message) kafka.Message {
	bytes, err := proto.Marshal(pb)
	if err != nil {
		panic(err)
	}

	return kafka.Message{
		Key:     nil,
		Value:   bytes,
		Headers: []kafka.Header{{Key: "X-Proto-Type", Value: []byte(pb.ProtoReflect().Descriptor().FullName())}},
		Time:    time.Time{},
	}
}
