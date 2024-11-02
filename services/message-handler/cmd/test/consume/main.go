package main

import (
	"context"
	"fmt"
	"github.com/emortalmc/proto-specs/gen/go/message/messagehandler"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/runtime/protoimpl"
)

const messagesTopic = "message-handler"

func main() {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   messagesTopic,
	})

	ctx := context.Background()

	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			panic(err)
		}

		var protoType string
		for _, header := range m.Headers {
			if header.Key == "X-Proto-Type" {
				protoType = string(header.Value)
			}
		}
		if protoType == "" {
			panic("no proto type found in message headers")
		}

		fmt.Printf("%s (%s): %s\n", m.Time, protoType, getMessageContents(protoType, m.Value))
	}
}

func getMessageContents(protoType string, b []byte) string {
	var parsedMsg protoreflect.ProtoMessage

	switch protoType {
	case string((&messagehandler.ChatMessageCreatedMessage{}).ProtoReflect().Descriptor().FullName()):
		parsedMsg = &messagehandler.ChatMessageCreatedMessage{}
	case string((&messagehandler.PrivateMessageCreatedMessage{}).ProtoReflect().Descriptor().FullName()):
		parsedMsg = &messagehandler.PrivateMessageCreatedMessage{}
	default:
		panic("unknown proto type " + protoType)
	}

	if err := proto.Unmarshal(b, parsedMsg); err != nil {
		panic(err)
	}

	return protoimpl.X.MessageStringOf(parsedMsg)
}
