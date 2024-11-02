package main

import (
	"context"
	"github.com/emortalmc/proto-specs/gen/go/message/common"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
	"log"
	"time"
)

const pIdStr = "8d36737e-1c0a-4a71-87de-9906f577845e"

func main() {
	w := &kafka.Writer{
		Addr:      kafka.TCP("localhost:9092"),
		Topic:     "mc-connections",
		BatchSize: 1,
		Async:     false,
	}

	connMsg := &common.PlayerConnectMessage{
		PlayerId:       pIdStr,
		PlayerUsername: "Expectational",
		ServerId:       "test-proxy-0000",
	}
	//
	//switchMsg := &common.PlayerSwitchServerMessage{
	//	PlayerID: pIdStr,
	//	ServerID: "test-server-1111",
	//}
	//
	//logoutMsg := &common.PlayerDisconnectMessage{
	//	PlayerID:       pIdStr,
	//	PlayerUsername: "Expectational",
	//}

	err := w.WriteMessages(context.Background(),
		createKafkaMessage(connMsg),
		//createKafkaMessage(switchMsg),
		//createKafkaMessage(logoutMsg),
	)
	if err != nil {
		panic(err)
	}

	log.Println("Message sent!")
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
