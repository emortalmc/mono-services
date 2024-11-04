package model

import (
	"github.com/emortalmc/mono-services/services/party-manager/internal/utils"
	"github.com/emortalmc/proto-specs/gen/go/model/common"
	pb "github.com/emortalmc/proto-specs/gen/go/model/party"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

type Party struct {
	ID primitive.ObjectID `bson:"_id"`

	EventID string `bson:"eventId,omitempty"`

	LeaderId uuid.UUID      `bson:"leaderId"`
	Members  []*PartyMember `bson:"members"`

	Open bool `bson:"open"`
}

func NewParty(leaderId uuid.UUID, leaderUsername string) *Party {
	return &Party{
		LeaderId: leaderId,
		Members:  []*PartyMember{{PlayerID: leaderId, Username: leaderUsername}},

		Open: false,
	}
}

func (p *Party) ToProto() *pb.Party {
	memberProtos := make([]*pb.PartyMember, len(p.Members))
	for i, member := range p.Members {
		memberProtos[i] = member.ToProto()
	}

	return &pb.Party{
		Id:       p.ID.Hex(),
		LeaderId: p.LeaderId.String(),
		Members:  memberProtos,
		Open:     p.Open,
	}
}

func (p *Party) ContainsMember(id uuid.UUID) bool {
	for _, member := range p.Members {
		if member.PlayerID == id {
			return true
		}
	}

	return false
}

func (p *Party) GetMember(id uuid.UUID) (member *PartyMember, ok bool) {
	for _, loopMember := range p.Members {
		if loopMember.PlayerID == id {
			member = loopMember
			ok = true
			return
		}
	}

	return
}

func (p *Party) IsEventParty() bool {
	return p.EventID != ""
}

type PartyMember struct {
	PlayerID uuid.UUID `bson:"playerId"`
	Username string    `bson:"username"`
}

func (m *PartyMember) ToProto() *pb.PartyMember {
	return &pb.PartyMember{
		Id:       m.PlayerID.String(),
		Username: m.Username,
	}
}

type PartyInvite struct {
	ID      primitive.ObjectID `bson:"_id"`
	PartyID primitive.ObjectID `bson:"partyId"`

	InviterID       uuid.UUID `bson:"inviterId"`
	InviterUsername string    `bson:"inviterUsername"`

	TargetID       uuid.UUID `bson:"targetId"`
	TargetUsername string    `bson:"targetUsername"`

	// ExpiresAt is the time at which the invite expires.
	// Note: This depends on an expiry index being created on MongoDB that reads the expiresAt field.
	ExpiresAt time.Time `bson:"expiresAt"`
}

func (i *PartyInvite) ToProto() *pb.PartyInvite {
	return &pb.PartyInvite{
		PartyId: i.PartyID.Hex(),

		TargetId:       i.TargetID.String(),
		TargetUsername: i.TargetUsername,

		SenderId:       i.InviterID.String(),
		SenderUsername: i.InviterUsername,

		ExpiresAt: timestamppb.New(i.ExpiresAt),
	}
}

type PartySettings struct {
	PlayerID uuid.UUID `bson:"_id"`

	DequeueOnDisconnect bool `bson:"dequeueOnDisconnect"`
	AllowMemberDequeue  bool `bson:"allowMemberDequeue"`
	AllowMemberInvite   bool `bson:"allowMemberInvite"`

	// InDB is whether the settings originated from the database or not.
	// If false, the default is probably returned.
	InDB bool `bson:"-"`
}

func NewPartySettings(playerId uuid.UUID) *PartySettings {
	return &PartySettings{
		PlayerID: playerId,

		DequeueOnDisconnect: false,
		AllowMemberDequeue:  false,
		AllowMemberInvite:   false,
	}
}

func (s *PartySettings) ToProto() *pb.PartySettings {
	return &pb.PartySettings{
		DequeueOnDisconnect: s.DequeueOnDisconnect,
		AllowMemberDequeue:  s.AllowMemberDequeue,
		AllowMemberInvite:   s.AllowMemberInvite,
	}
}

type PlayerSkin struct {
	Texture   string `bson:"texture"`
	Signature string `bson:"signature"`
}

type Event struct {
	ID string `bson:"_id"`

	OwnerID       uuid.UUID  `bson:"ownerId"`
	OwnerUsername string     `bson:"ownerUsername"`
	Skin          PlayerSkin `bson:"skin"`

	DisplayTime *time.Time `bson:"displayTime"`
	StartTime   *time.Time `bson:"startTime"`

	// PartyID only present if the event has started.
	PartyID *primitive.ObjectID `bson:"partyId,omitempty"`
}

func (e *Event) ToProto() *pb.EventData {
	var displayTime, startTime *timestamppb.Timestamp
	if e.DisplayTime != nil {
		displayTime = timestamppb.New(*e.DisplayTime)
	}
	if e.StartTime != nil {
		startTime = timestamppb.New(*e.StartTime)
	}

	var partyId *string
	if e.PartyID != nil {
		partyId = utils.PointerOf(e.PartyID.Hex())
	}

	return &pb.EventData{
		Id:            e.ID,
		OwnerId:       e.OwnerID.String(),
		OwnerUsername: e.OwnerUsername,
		OwnerSkin: &common.PlayerSkin{
			Texture:   e.Skin.Texture,
			Signature: e.Skin.Signature,
		},
		DisplayTime: displayTime,
		StartTime:   startTime,
		PartyId:     partyId,
	}
}
