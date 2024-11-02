package main

import (
	"context"
	"fmt"
	"github.com/emortalmc/proto-specs/gen/go/message/common"
	"github.com/emortalmc/proto-specs/gen/go/model/messagehandler"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

const messagesTopic = "mc-messages"

func main() {
	w := &kafka.Writer{
		Addr:     kafka.TCP("localhost:9092"),
		Topic:    messagesTopic,
		Balancer: &kafka.LeastBytes{},
	}

	msg := common.PlayerChatMessageMessage{Message: &messagehandler.ChatMessage{
		SenderId:       "8d36737e-1c0a-4a71-87de-9906f577845e",
		SenderUsername: "Expectational",
		Message:        "Test chat message",
	}}

	bytes, err := proto.Marshal(&msg)
	if err != nil {
		panic(fmt.Errorf("failed to marshal proto to bytes: %s", err))
	}

	if err := w.WriteMessages(context.Background(), kafka.Message{
		Value:   bytes,
		Headers: []kafka.Header{{Key: "X-Proto-Type", Value: []byte(msg.ProtoReflect().Descriptor().FullName())}},
	}); err != nil {
		panic(fmt.Errorf("failed to write message: %s", err))
	}

	fmt.Println("Message sent! " + msg.String())
}
