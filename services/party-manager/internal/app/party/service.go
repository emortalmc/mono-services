package party

import (
	"context"
	"errors"
	"fmt"
	"github.com/emortalmc/mono-services/services/party-manager/internal/repository/model"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"time"
)

type Service struct {
	log   *zap.SugaredLogger
	repo  ReadWriter
	notif KafkaWriter
}

func NewService(log *zap.SugaredLogger, repo ReadWriter, notif KafkaWriter) *Service {
	return &Service{
		log:   log,
		repo:  repo,
		notif: notif,
	}
}

var ErrPartyLeaderNotFound = errors.New("party leader not found")

func (s *Service) EmptyParty(ctx context.Context, party *model.Party, partyOpen bool) error {
	leader, ok := party.GetMember(party.LeaderId)
	if !ok {
		return ErrPartyLeaderNotFound
	}

	if err := s.repo.SetPartyMembers(ctx, party.ID, []*model.PartyMember{leader}); err != nil {
		return fmt.Errorf("failed to set party members: %w", err)
	}

	if err := s.repo.DeletePartyInvitesByPartyId(ctx, party.ID); err != nil {
		return fmt.Errorf("failed to delete party invites: %w", err)
	}

	if party.Open != partyOpen {
		if err := s.repo.SetPartyOpen(ctx, party.ID, partyOpen); err != nil {
			return fmt.Errorf("failed to set party open: %w", err)
		}
	}

	s.notif.PartyEmptied(ctx, party)

	// Go through all the members and make them a new party.
	for _, member := range party.Members {
		if member.PlayerID == party.LeaderId {
			continue
		}

		newParty := model.NewParty(member.PlayerID, member.Username)
		if err := s.repo.CreateParty(ctx, newParty); err != nil {
			return fmt.Errorf("failed to create party: %w", err)
		}

		s.notif.PartyCreated(ctx, newParty)
	}

	return nil
}

func (s *Service) SetOpenParty(ctx context.Context, partyID primitive.ObjectID, open bool) error {
	if err := s.repo.SetPartyOpen(ctx, partyID, open); err != nil {
		return fmt.Errorf("failed to set party open: %w", err)
	}

	s.notif.PartyOpenChanged(ctx, partyID, open)

	return nil
}

var ErrExecutorNotLeader = errors.New("executor is not the leader of the party")
var ErrTargetAndSelfSameParty = errors.New("target is in the same party as the executor")
var ErrTargetAlreadyInvited = errors.New("target is already invited to the party")

func (s *Service) InvitePlayer(ctx context.Context, executorID uuid.UUID, executorUsername string, targetID uuid.UUID,
	targetUsername string) (*model.PartyInvite, error) {

	// Get party of the inviter
	party, err := s.repo.GetPartyByMemberID(ctx, executorID)
	if err != nil {
		// NOTE: We no longer implicitly create a party because a player should always be in a party
		return nil, err
	}

	settings, err := s.GetPartySettings(ctx, party.LeaderId)
	if err != nil {
		return nil, err
	}

	if party.LeaderId != executorID {
		if !settings.AllowMemberInvite {
			return nil, ErrExecutorNotLeader
		}
	}

	targetParty, err := s.repo.GetPartyByMemberID(ctx, targetID)
	if err != nil {
		return nil, err
	}

	if targetParty.ID == party.ID {
		return nil, ErrTargetAndSelfSameParty
	}

	// Check if the target is already invited
	inviteExists, err := s.repo.DoesPartyInviteExist(ctx, party.ID, targetID)
	if err != nil {
		return nil, fmt.Errorf("failed to check if party invite exists: %w", err)
	}
	if inviteExists {
		return nil, ErrTargetAlreadyInvited
	}

	// Create invite
	invite := &model.PartyInvite{
		PartyID: party.ID,

		InviterID:       executorID,
		InviterUsername: executorUsername,

		TargetID:       targetID,
		TargetUsername: targetUsername,

		ExpiresAt: time.Now().In(time.UTC).Add(5 * time.Minute),
	}

	// Save invite
	err = s.repo.CreatePartyInvite(ctx, invite)
	if err != nil {
		return nil, fmt.Errorf("failed to create party invite: %w", err)
	}

	s.notif.PartyInviteCreated(ctx, invite)

	return invite, nil
}

var ErrNotInvited = errors.New("player is not invited to the party")
var ErrAlreadyInSameParty = errors.New("player is already in the same party")

func (s *Service) JoinPartyByMemberID(ctx context.Context, playerID uuid.UUID, playerUsername string, targetID uuid.UUID) (*model.Party, error) {
	currentParty, err := s.repo.GetPartyByMemberID(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get party by member id: %w", err)
	}

	targetParty, err := s.repo.GetPartyByMemberID(ctx, targetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get party by member id: %w", err)
	}

	if targetParty.ID == currentParty.ID {
		return nil, ErrAlreadyInSameParty
	}

	if !targetParty.Open {
		if err := s.repo.DeletePartyInvite(ctx, targetParty.ID, playerID); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, ErrNotInvited
			} else {
				return nil, fmt.Errorf("failed to delete party invite: %w", err)
			}
		}
	}

	if len(currentParty.Members) == 1 {
		if err := s.repo.DeleteParty(ctx, currentParty.ID); err != nil {
			return nil, fmt.Errorf("failed to delete party: %w", err)
		}

		s.notif.PartyDeleted(ctx, currentParty)
	} else { // RemovePlayerFromParty handles leader election if they are the leader of the party
		if _, err := s.RemovePlayerFromParty(ctx, playerID, false); err != nil {
			return nil, fmt.Errorf("failed to remove player from party: %w", err)
		}
	}

	newPartyMember := &model.PartyMember{
		PlayerID: playerID,
		Username: playerUsername,
	}

	if err := s.repo.AddPartyMember(ctx, targetParty.ID, newPartyMember); err != nil {
		return nil, fmt.Errorf("failed to add party member: %w", err)
	}

	s.notif.PartyPlayerJoined(ctx, targetParty.ID, newPartyMember)

	return targetParty, nil
}

var ErrPlayerIsOnlyMember = errors.New("player is the only member of the party")

func (s *Service) RemovePlayerFromParty(ctx context.Context, playerID uuid.UUID, createNewParty bool) (*model.Party, error) {
	party, err := s.repo.GetPartyByMemberID(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get party by member id: %w", err)
	}

	if len(party.Members) == 1 {
		return party, ErrPlayerIsOnlyMember
	}

	if err := s.repo.RemoveMemberFromParty(ctx, party.ID, playerID); err != nil {
		return nil, fmt.Errorf("failed to remove party member: %w", err) // wrapped is of type ErrNotInParty
	}

	member, ok := party.GetMember(playerID)
	if !ok {
		return nil, fmt.Errorf("failed to get party member: %w", err)
	}

	s.notif.PartyPlayerLeft(ctx, party.ID, member)

	if party.LeaderId == playerID {
		newLeader, err := electNewPartyLeader(party)
		if err != nil {
			return nil, fmt.Errorf("failed to elect new party leader: %w", err)
		}

		if err := s.repo.SetPartyLeader(ctx, party.ID, newLeader.PlayerID); err != nil {
			return nil, fmt.Errorf("failed to set party leader: %w", err)
		}

		s.notif.PartyLeaderChanged(ctx, party.ID, newLeader)
	}

	if createNewParty {
		if party, err = s.putPlayerInNewParty(ctx, playerID, member.Username); err != nil {
			return nil, fmt.Errorf("failed to put player in new party: %w", err)
		}
	}

	return party, nil
}

var ErrTargetNotInParty = errors.New("target is not in the party")

// todo might be an issue here with someone trying to kick themselves?
func (s *Service) KickPlayerFromParty(ctx context.Context, executorID uuid.UUID, targetID uuid.UUID) error {
	party, err := s.repo.GetPartyByMemberID(ctx, executorID)
	if err != nil {
		return fmt.Errorf("failed to get party by member id: %w", err)
	}

	if party.LeaderId != executorID {
		return ErrExecutorNotLeader
	}

	targetMember, ok := party.GetMember(targetID)
	if !ok {
		return ErrTargetNotInParty
	}

	// All checks passed, remove the player from the party

	if _, err := s.RemovePlayerFromParty(ctx, targetID, true); err != nil {
		return fmt.Errorf("failed to remove player from party: %w", err)
	}

	executorMember, ok := party.GetMember(executorID)
	if !ok {
		return fmt.Errorf("failed to get party member as they are not in own party?: %w", err)
	}

	s.notif.PartyPlayerKicked(ctx, party.ID, targetMember, executorMember)

	if _, err := s.putPlayerInNewParty(ctx, targetID, targetMember.Username); err != nil {
		return fmt.Errorf("failed to put player in new party: %w", err)
	}

	return nil
}

var ErrPlayerAlreadyLeader = errors.New("player is already the leader of the party")

func (s *Service) ChangePartyLeader(ctx context.Context, party *model.Party, newLeaderID uuid.UUID) error {
	leaderMember, ok := party.GetMember(newLeaderID)
	if !ok {
		return ErrTargetNotInParty
	}

	if party.LeaderId == newLeaderID {
		return ErrPlayerAlreadyLeader
	}

	if err := s.repo.SetPartyLeader(ctx, party.ID, newLeaderID); err != nil {
		return fmt.Errorf("failed to set party leaderMember: %w", err)
	}

	s.notif.PartyLeaderChanged(ctx, party.ID, leaderMember)

	return nil
}

func (s *Service) putPlayerInNewParty(ctx context.Context, playerID uuid.UUID, playerUsername string) (*model.Party, error) {
	party := model.NewParty(playerID, playerUsername)

	if err := s.repo.CreateParty(ctx, party); err != nil {
		return nil, fmt.Errorf("failed to create party: %w", err)
	}

	s.notif.PartyCreated(ctx, party)

	return party, nil
}

// GetPartySettings gets the party settings for a player or returns the default settings if the player has none.
func (s *Service) GetPartySettings(ctx context.Context, playerId uuid.UUID) (*model.PartySettings, error) {
	settings, err := s.repo.GetPartySettings(ctx, playerId)
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("failed to get party settings: %w", err)
		}
		return model.NewPartySettings(playerId), nil
	}

	return settings, nil
}
