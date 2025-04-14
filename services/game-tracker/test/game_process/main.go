package main

import (
	"context"
	"fmt"
	"github.com/emortalmc/mono-services/services/game-tracker/internal/config"
	"github.com/emortalmc/mono-services/services/game-tracker/internal/utils"
	"github.com/emortalmc/proto-specs/gen/go/message/gametracker"
	pbmodel "github.com/emortalmc/proto-specs/gen/go/model/gametracker"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"log"
	"math/rand"
	"time"
)

var (
	serverId = "tower-defence-test-" + uuid.New().String()[:6]
	gameId   = primitive.NewObjectID()

	expectationalId = uuid.MustParse("8d36737e-1c0a-4a71-87de-9906f577845e")
	emortaldevId    = uuid.MustParse("7bd5b459-1e6b-4753-8274-1fbd2fe9a4d5")

	defaultPlayers = []*pbmodel.BasicGamePlayer{
		{
			Id:       expectationalId.String(),
			Username: "Expectational",
		},
		{
			Id:       emortaldevId.String(),
			Username: "emortaldev",
		},
	}
)

type app struct {
	w *kafka.Writer
}

func main() {
	cfg := config.LoadGlobalConfig()

	w := &kafka.Writer{
		Addr:     kafka.TCP(fmt.Sprintf("%s:%d", cfg.Kafka.Host, cfg.Kafka.Port)),
		Topic:    "games",
		Balancer: &kafka.LeastBytes{},
		Async:    false,
	}

	a := &app{
		w: w,
	}

	a.writeStartMessage()
	log.Printf("wrote start message, waiting 5 seconds before sending update")
	time.Sleep(5 * time.Second)

	a.writeUpdateMessage()
	log.Printf("wrote update message, waiting 5 seconds before sending end")
	time.Sleep(5 * time.Second)

	a.writeFinishedMessage()
	log.Printf("wrote finished message. All done :)")
}

func (a *app) writeStartMessage() {
	tdStartData, err := anypb.New(&pbmodel.TowerDefenceStartData{
		HealthData: &pbmodel.TowerDefenceHealthData{
			MaxHealth:  1000,
			BlueHealth: 1000,
			RedHealth:  1000,
		},
	})
	if err != nil {
		panic(err)
	}

	teamStartData, err := anypb.New(&pbmodel.CommonGameTeamData{Teams: []*pbmodel.Team{
		{
			Id:           "blue",
			FriendlyName: "Blue",
			Color:        0x0000FF,
			PlayerIds:    []string{expectationalId.String()},
		},
		{
			Id:           "red",
			FriendlyName: "Red",
			Color:        0xFF0000,
			PlayerIds:    []string{emortaldevId.String()},
		},
	}})
	if err != nil {
		panic(err)
	}

	message := &gametracker.GameStartMessage{
		CommonData: &gametracker.CommonGameData{
			GameModeId: "tower-defence",
			GameId:     gameId.Hex(),
			ServerId:   serverId,
			Players:    defaultPlayers,
		},
		StartTime: timestamppb.Now(),
		MapId:     utils.Pointer("test-map"),
		Content:   []*anypb.Any{tdStartData, teamStartData},
	}

	a.writeMessages(message)
}

func (a *app) writeUpdateMessage() {
	tdUpdateData, err := anypb.New(&pbmodel.TowerDefenceUpdateData{
		HealthData: &pbmodel.TowerDefenceHealthData{
			MaxHealth:  1000,
			BlueHealth: 750,
			RedHealth:  500,
		},
	})
	if err != nil {
		panic(err)
	}

	message := &gametracker.GameUpdateMessage{
		CommonData: &gametracker.CommonGameData{
			GameModeId: "tower-defence",
			GameId:     gameId.Hex(),
			ServerId:   serverId,
			Players:    defaultPlayers,
		},
		Content: []*anypb.Any{tdUpdateData},
	}

	a.writeMessages(message)
}

func (a *app) writeFinishedMessage() {
	tdFinishData, err := anypb.New(&pbmodel.TowerDefenceFinishData{
		HealthData: &pbmodel.TowerDefenceHealthData{
			MaxHealth:  rand.Int31n(1000),
			BlueHealth: rand.Int31n(1000),
			RedHealth:  rand.Int31n(1000),
		},
	})
	if err != nil {
		panic(err)
	}

	winnerData, err := anypb.New(&pbmodel.CommonGameFinishWinnerData{
		WinnerIds: []string{defaultPlayers[0].GetId()},
		LoserIds:  []string{defaultPlayers[1].GetId()},
	})
	if err != nil {
		panic(err)
	}

	message := &gametracker.GameFinishMessage{
		CommonData: &gametracker.CommonGameData{
			GameModeId: "tower-defence",
			GameId:     gameId.Hex(),
			ServerId:   serverId,
			Players:    defaultPlayers,
		},
		EndTime: timestamppb.New(time.Now().Add(1 * time.Minute)),
		Content: []*anypb.Any{tdFinishData, winnerData},
	}

	a.writeMessages(message)
}

func (a *app) writeMessages(messages ...proto.Message) {
	kMessages := make([]kafka.Message, len(messages))

	for i, m := range messages {
		bytes, err := proto.Marshal(m)
		if err != nil {
			panic(err)
		}

		kMessages[i] = kafka.Message{
			Headers: []kafka.Header{
				{
					Key:   "X-Proto-Type",
					Value: []byte(m.ProtoReflect().Descriptor().FullName()),
				},
			},
			Value: bytes,
		}
	}

	if err := a.w.WriteMessages(context.Background(), kMessages...); err != nil {
		panic(err)
	}
}
