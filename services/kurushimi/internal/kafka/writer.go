package kafka

import (
	"context"
	"fmt"
	"github.com/emortalmc/mono-services/services/kurushimi/internal/config"
	"github.com/emortalmc/mono-services/services/kurushimi/internal/repository/model"
	msg "github.com/emortalmc/proto-specs/gen/go/message/matchmaker"
	pb "github.com/emortalmc/proto-specs/gen/go/model/matchmaker"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"sync"
	"time"
)

const writeTopic = "matchmaker"

// TODO fire these methods
type Notifier interface {
	TicketCreated(ctx context.Context, ticket *model.Ticket) error
	TicketUpdated(ctx context.Context, ticket *model.Ticket) error
	TicketDeleted(ctx context.Context, ticket *pb.Ticket, reason msg.TicketDeletedMessage_Reason) error

	PendingMatchCreated(ctx context.Context, match *model.PendingMatch) error
	PendingMatchUpdated(ctx context.Context, match *model.PendingMatch) error
	PendingMatchDeleted(ctx context.Context, match *model.PendingMatch, reason msg.PendingMatchDeletedMessage_Reason) error

	MatchCreated(ctx context.Context, match *pb.Match) error
}

type kafkaNotifier struct {
	w *kafka.Writer
}

func NewKafkaNotifier(ctx context.Context, wg *sync.WaitGroup, config config.KafkaConfig, logger *zap.SugaredLogger) Notifier {
	w := &kafka.Writer{
		Addr:         kafka.TCP(fmt.Sprintf("%s:%d", config.Host, config.Port)),
		Topic:        writeTopic,
		Balancer:     &kafka.LeastBytes{},
		Async:        true,
		BatchTimeout: 50 * time.Millisecond,
		ErrorLogger:  kafka.LoggerFunc(logger.Errorw),
	}

	wg.Add(1)
	go func() {
		<-ctx.Done()
		if err := w.Close(); err != nil {
			logger.Errorw("failed to close kafka writer", "error", err)
		}
		wg.Done()
	}()

	return &kafkaNotifier{w: w}
}

func (k *kafkaNotifier) TicketCreated(ctx context.Context, ticket *model.Ticket) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	pMsg := &msg.TicketCreatedMessage{Ticket: ticket.ToProto()}
	bytes, err := proto.Marshal(pMsg)
	if err != nil {
		return err
	}

	err = k.w.WriteMessages(ctx, kafka.Message{
		Headers: []kafka.Header{{Key: "X-Proto-Type", Value: []byte(pMsg.ProtoReflect().Descriptor().FullName())}},
		Value:   bytes,
	})

	return err
}

func (k *kafkaNotifier) TicketUpdated(ctx context.Context, ticket *model.Ticket) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	pMsg := &msg.TicketUpdatedMessage{NewTicket: ticket.ToProto()}
	bytes, err := proto.Marshal(pMsg)
	if err != nil {
		return err
	}

	err = k.w.WriteMessages(ctx, kafka.Message{
		Headers: []kafka.Header{{Key: "X-Proto-Type", Value: []byte(pMsg.ProtoReflect().Descriptor().FullName())}},
		Value:   bytes,
	})

	return err
}

func (k *kafkaNotifier) TicketDeleted(ctx context.Context, ticket *pb.Ticket, reason msg.TicketDeletedMessage_Reason) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	pMsg := &msg.TicketDeletedMessage{Ticket: ticket, Reason: reason}
	bytes, err := proto.Marshal(pMsg)
	if err != nil {
		return err
	}

	err = k.w.WriteMessages(ctx, kafka.Message{
		Headers: []kafka.Header{{Key: "X-Proto-Type", Value: []byte(pMsg.ProtoReflect().Descriptor().FullName())}},
		Value:   bytes,
	})

	return err
}

func (k *kafkaNotifier) PendingMatchCreated(ctx context.Context, match *model.PendingMatch) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	pMsg := &msg.PendingMatchCreatedMessage{PendingMatch: match.ToProto()}
	bytes, err := proto.Marshal(pMsg)
	if err != nil {
		return err
	}

	err = k.w.WriteMessages(ctx, kafka.Message{
		Headers: []kafka.Header{{Key: "X-Proto-Type", Value: []byte(pMsg.ProtoReflect().Descriptor().FullName())}},
		Value:   bytes,
	})

	return err
}

func (k *kafkaNotifier) PendingMatchUpdated(ctx context.Context, match *model.PendingMatch) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	pMsg := &msg.PendingMatchUpdatedMessage{PendingMatch: match.ToProto()}
	bytes, err := proto.Marshal(pMsg)
	if err != nil {
		return err
	}

	err = k.w.WriteMessages(ctx, kafka.Message{
		Headers: []kafka.Header{{Key: "X-Proto-Type", Value: []byte(pMsg.ProtoReflect().Descriptor().FullName())}},
		Value:   bytes,
	})

	return err
}

func (k *kafkaNotifier) PendingMatchDeleted(ctx context.Context, match *model.PendingMatch, reason msg.PendingMatchDeletedMessage_Reason) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	pMsg := &msg.PendingMatchDeletedMessage{PendingMatch: match.ToProto(), Reason: reason}
	bytes, err := proto.Marshal(pMsg)
	if err != nil {
		return err
	}

	err = k.w.WriteMessages(ctx, kafka.Message{
		Headers: []kafka.Header{{Key: "X-Proto-Type", Value: []byte(pMsg.ProtoReflect().Descriptor().FullName())}},
		Value:   bytes,
	})

	return err
}

func (k *kafkaNotifier) MatchCreated(ctx context.Context, match *pb.Match) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	pMsg := &msg.MatchCreatedMessage{Match: match}
	bytes, err := proto.Marshal(pMsg)
	if err != nil {
		return err
	}

	err = k.w.WriteMessages(ctx, kafka.Message{
		Headers: []kafka.Header{{Key: "X-Proto-Type", Value: []byte(pMsg.ProtoReflect().Descriptor().FullName())}},
		Value:   bytes,
	})

	return err
}
