package party

import (
	"context"
	"errors"
	"github.com/emortalmc/mono-services/services/party-manager/internal/repository/model"
	"github.com/google/uuid"
	"math/rand"
)

func (s *Service) HandlePlayerDisconnect(ctx context.Context, playerID uuid.UUID) {
	party, err := s.repo.GetPartyByMemberID(ctx, playerID)
	if err != nil {
		s.log.Errorw("failed to get party by member id", err)
		return
	}

	// IF it is an event party then we don't want any leaving, promotion, deleting, ets.. logic to run
	if party.IsEventParty() && party.LeaderId == playerID {
		return
	}

	// Party only has the leader in it
	if playerID == party.LeaderId && len(party.Members) < 2 {
		if err := s.repo.DeleteParty(ctx, party.ID); err != nil {
			s.log.Errorw("failed to delete party", err)
			return
		}

		if err := s.repo.DeletePartyInvitesByPartyId(ctx, party.ID); err != nil {
			s.log.Errorw("failed to delete party invites", err)
			return
		}

		s.notif.PartyDeleted(ctx, party)
		return
	}

	// Player may still be the party leader, but it has more than 1 member OR is an event party.
	// As a result, do a new leader election.

	if err := s.repo.RemoveMemberFromParty(ctx, party.ID, playerID); err != nil {
		s.log.Errorw("failed to remove member from party", err)
		return
	}

	partyMember, ok := party.GetMember(playerID)
	if !ok {
		s.log.Errorw("party member not found", "partyID", party.ID, "playerID", playerID)
		return
	}

	s.notif.PartyPlayerLeft(ctx, party.ID, partyMember)

	// If the player is the current leader, elect a new leader
	if playerID == party.LeaderId {
		newLeader, err := electNewPartyLeader(party)
		if err != nil {
			s.log.Errorw("failed to elect new party leader", err)
			return // todo don't just return here - it leaves the party in a broken state. Very bad.
		}

		if err := s.repo.SetPartyLeader(ctx, party.ID, newLeader.PlayerID); err != nil {
			s.log.Errorw("failed to set party leader", err)
			return
		}

		s.notif.PartyLeaderChanged(ctx, party.ID, newLeader)
	}
}

func (s *Service) HandlePlayerConnect(ctx context.Context, playerId uuid.UUID, username string) {
	if party, err := s.repo.GetPartyByMemberID(ctx, playerId); err == nil {
		if party.IsEventParty() && party.LeaderId == playerId {
			return
		}

		s.log.Errorw("player already in party??", "playerID", playerId, "party", party)
		return
	}

	party := model.NewParty(playerId, username)

	if err := s.repo.CreateParty(ctx, party); err != nil {
		s.log.Errorw("failed to create party", err)
		return
	}

	s.notif.PartyCreated(ctx, party)
}

var ErrLeaderElecNotEnoughMembers = errors.New("party member count must be >= 2")

// electNewPartyLeader elects a member of the party as a new leader of the party and returns the member that is the new
// leader. It assumes that party.LeaderId is still current to ignore them as a candidate for the new leader.
func electNewPartyLeader(party *model.Party) (*model.PartyMember, error) {
	if len(party.Members) < 2 {
		return nil, ErrLeaderElecNotEnoughMembers
	}

	members := make([]*model.PartyMember, len(party.Members)-1)

	i := 0
	for _, member := range party.Members {
		if member.PlayerID == party.LeaderId {
			continue
		}

		members[i] = member
		i++
	}

	newLeader := members[rand.Intn(len(members))]
	return newLeader, nil
}
