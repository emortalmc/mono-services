package model

import (
	commonmodel "github.com/emortalmc/proto-specs/gen/go/model/common"
	"github.com/emortalmc/proto-specs/gen/go/model/mcplayer"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

type Player struct {
	ID              uuid.UUID  `bson:"_id"`
	CurrentUsername string     `bson:"currentUsername"`
	CurrentSkin     PlayerSkin `bson:"currentSkin,omitempty"` // Only null as it didn't use to be stored

	FirstLogin time.Time `bson:"firstLogin"`

	// LastOnline this is only updated when the player logs out/in. If the player is CurrentlyOnline, trust that LastOnline is time.Now()
	LastOnline    time.Time     `bson:"lastOnline"`
	TotalPlaytime time.Duration `bson:"totalPlaytime"`

	// Badges IDs of the badges the player has
	Badges []string `bson:"badges,omitempty"`

	// ActiveBadge ID of the badge the player has currently active (nil if none)
	ActiveBadge *string `bson:"activeBadge,omitempty"`

	CurrentServer *CurrentServer `bson:"currentServer,omitempty"`

	Experience int64 `bson:"experience"`
}

func (p Player) IsEmpty() bool {
	return p.ID == uuid.Nil
}

func (p Player) ToProto(session LoginSession) *mcplayer.McPlayer {
	return &mcplayer.McPlayer{
		Id:               p.ID.String(),
		CurrentUsername:  p.CurrentUsername,
		FirstLogin:       timestamppb.New(p.FirstLogin),
		LastOnline:       timestamppb.New(p.LastOnline),
		CurrentlyOnline:  p.CurrentServer != nil,
		CurrentSession:   session.ToProto(),
		HistoricPlayTime: durationpb.New(p.TotalPlaytime),
		CurrentServer:    p.CurrentServer.ToProto(),
		CurrentSkin:      p.CurrentSkin.ToProto(),
		Experience:       uint64(p.Experience),
	}
}

var OnlinePlayerProjection = map[string]interface{}{
	"_id":             1,
	"currentUsername": 1,
	"currentServer":   1,
}

// OnlinePlayer a partial player object that is used
// by the PlayerTracker to track online players
type OnlinePlayer struct {
	ID              uuid.UUID `bson:"_id"`
	CurrentUsername string    `bson:"currentUsername"`

	CurrentServer *CurrentServer `bson:"currentServer,omitempty"`
}

func (p OnlinePlayer) ToProto() *mcplayer.OnlinePlayer {
	return &mcplayer.OnlinePlayer{
		PlayerId: p.ID.String(),
		Username: p.CurrentUsername,
		Server:   p.CurrentServer.ToProto(),
	}
}

var BadgePlayerProjection = map[string]interface{}{
	"_id":         1,
	"badges":      1,
	"activeBadge": 1,
}

type BadgePlayer struct {
	ID uuid.UUID `bson:"_id"`

	// BadgeIDs IDs of the badge the player has
	BadgeIDs []string `bson:"badges,omitempty"`

	// ActiveBadge ID of the badge the player has currently active (nil if none)
	ActiveBadge *string `bson:"activeBadge,omitempty"`
}

type CurrentServer struct {
	ServerID  string `bson:"serverId"`
	ProxyID   string `bson:"proxyId"`
	FleetName string `bson:"fleetName"`
}

func (s *CurrentServer) ToProto() *mcplayer.CurrentServer {
	if s == nil {
		return nil
	}

	return &mcplayer.CurrentServer{
		ServerId:  s.ServerID,
		ProxyId:   s.ProxyID,
		FleetName: s.FleetName,
	}
}

type PlayerSkin struct {
	Texture   string `bson:"texture"`
	Signature string `bson:"signature"`
}

func (s PlayerSkin) IsEmpty() bool {
	return s.Texture == ""
}

func PlayerSkinFromProto(s *commonmodel.PlayerSkin) PlayerSkin {
	if s == nil {
		return PlayerSkin{}
	}

	return PlayerSkin{
		Texture:   s.Texture,
		Signature: s.Signature,
	}
}

func (s PlayerSkin) ToProto() *commonmodel.PlayerSkin {
	if s.IsEmpty() {
		return nil
	}

	return &commonmodel.PlayerSkin{
		Texture:   s.Texture,
		Signature: s.Signature,
	}
}

type LoginSession struct {
	ID       primitive.ObjectID `bson:"_id"`
	PlayerID uuid.UUID          `bson:"playerId"`

	LogoutTime *time.Time `bson:"logoutTime,omitempty"`
}

func (s LoginSession) IsEmpty() bool {
	return s.ID == primitive.NilObjectID
}

func (s LoginSession) GetDuration() time.Duration {
	if s.LogoutTime == nil {
		return time.Since(s.ID.Timestamp())
	}

	return s.LogoutTime.Sub(s.ID.Timestamp())
}

func (s LoginSession) ToProto() *mcplayer.LoginSession {
	if s.IsEmpty() {
		return nil
	}

	proto := &mcplayer.LoginSession{
		SessionId: s.ID.String(),
		LoginTime: timestamppb.New(s.ID.Timestamp()),
	}

	if s.LogoutTime != nil {
		proto.LogoutTime = timestamppb.New(*s.LogoutTime)
	}

	return proto
}

type PlayerUsername struct {
	ID       primitive.ObjectID `bson:"_id"`
	PlayerID uuid.UUID          `bson:"playerId"`
	Username string             `bson:"username"`
}

type ExperienceTransaction struct {
	ID       primitive.ObjectID `bson:"_id"`
	PlayerID uuid.UUID          `bson:"playerId"`
	Amount   int64              `bson:"amount"`
	Reason   string             `bson:"reason"`
}
