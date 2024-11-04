package grpc

//
//import (
//	"context"
//	pb "github.com/emortalmc/proto-specs/gen/go/grpc/party"
//	pbmodel "github.com/emortalmc/proto-specs/gen/go/model/party"
//	"github.com/golang/mock/gomock"
//	"github.com/google/uuid"
//	"github.com/stretchr/testify/assert"
//	"go.mongodb.org/mongo-driver/bson/primitive"
//	"go.mongodb.org/mongo-driver/mongo"
//	"github.com/emortalmc/mono-services/services/party-manager/internal/kafka/writer"
//	"github.com/emortalmc/mono-services/services/party-manager/internal/repository"
//	"github.com/emortalmc/mono-services/services/party-manager/internal/repository/model"
//	"testing"
//	"time"
//)
//
//func TestPartyService_EmptyParty(t *testing.T) {
//	partyId := primitive.NewObjectID()
//	memberId := uuid.New()
//
//	test := []struct {
//		name string
//		req  *pb.EmptyPartyRequest
//
//		getPartyByIdReq primitive.ObjectID
//		getPartyByIdRes *model.Party
//		getPartyByIdErr error
//
//		getPartyByMemberIdReq uuid.UUID
//		getPartyByMemberIdRes *model.Party
//		getPartyByMemberIdErr error
//
//		deletePartyReq primitive.ObjectID
//		deletePartyErr error
//
//		deletePartyInvitesByPartyIdReq bool
//		deletePartyInvitesByPartyIdErr error
//
//		wantErr error
//	}{
//		{
//			name: "success_by_party_id",
//			req: &pb.EmptyPartyRequest{
//				Id: &pb.EmptyPartyRequest_PartyId{PartyId: partyId.Hex()},
//			},
//
//			getPartyByIdReq: partyId,
//			getPartyByIdRes: &model.Party{ID: partyId, LeaderId: memberId, Members: []*model.PartyMember{{PlayerID: memberId, Username: "test"}}},
//
//			deletePartyReq: partyId,
//
//			deletePartyInvitesByPartyIdReq: true,
//		},
//		{
//			name: "success_by_member_id",
//			req: &pb.EmptyPartyRequest{
//				Id: &pb.EmptyPartyRequest_PlayerId{PlayerId: memberId.String()},
//			},
//
//			getPartyByMemberIdReq: memberId,
//			getPartyByMemberIdRes: &model.Party{ID: partyId, LeaderId: memberId, Members: []*model.PartyMember{{PlayerID: memberId, Username: "test"}}},
//
//			deletePartyReq: partyId,
//
//			deletePartyInvitesByPartyIdReq: true,
//		},
//	}
//
//	for _, tt := range test {
//		t.Run(tt.name, func(t *testing.T) {
//			ctx := context.Background()
//
//			// Create mock r
//			mockCntrl := gomock.NewController(t)
//			defer mockCntrl.Finish()
//
//			repo := repository.NewMockRepository(mockCntrl)
//
//			if tt.req.GetPartyId() != "" {
//				repo.EXPECT().GetPartyById(ctx, tt.getPartyByIdReq).Return(tt.getPartyByIdRes, tt.getPartyByIdErr)
//			}
//			if tt.getPartyByMemberIdReq != uuid.Nil {
//				repo.EXPECT().GetPartyByMemberId(ctx, tt.getPartyByMemberIdReq).Return(tt.getPartyByMemberIdRes, tt.getPartyByMemberIdErr)
//			}
//			if tt.deletePartyReq != primitive.NilObjectID {
//				var party *model.Party
//				if tt.getPartyByIdRes != nil {
//					party = tt.getPartyByIdRes
//				} else if tt.getPartyByMemberIdRes != nil {
//					party = tt.getPartyByMemberIdRes
//				}
//
//				leader, ok := party.GetMember(party.LeaderId)
//
//				if ok {
//					repo.EXPECT().SetPartyMembers(ctx, tt.deletePartyReq, []*model.PartyMember{leader}).Return(tt.deletePartyErr)
//				}
//			}
//			if tt.deletePartyInvitesByPartyIdReq {
//				repo.EXPECT().DeletePartyInvitesByPartyId(ctx, tt.deletePartyReq).Return(tt.deletePartyInvitesByPartyIdErr)
//			}
//
//			notif := writer.NewMockNotifier(mockCntrl)
//			// If there are no errors yet, expect a notification
//			if tt.deletePartyErr == nil && tt.getPartyByIdErr == nil && tt.getPartyByMemberIdErr == nil {
//				var party *model.Party
//				if tt.getPartyByIdRes != nil {
//					party = tt.getPartyByIdRes
//				} else if tt.getPartyByMemberIdRes != nil {
//					party = tt.getPartyByMemberIdRes
//				}
//				notif.EXPECT().PartyEmptied(ctx, party)
//			}
//
//			s := newPartyService(notif, repo)
//
//			res, err := s.EmptyParty(ctx, tt.req)
//			assert.Equalf(t, tt.wantErr, err, "wantErr: %+v, got: %+v", tt.wantErr, err)
//			assert.Empty(t, res)
//		})
//	}
//}
//
//func TestPartyService_GetParty(t *testing.T) {
//	partyId := primitive.NewObjectID()
//	memberId := uuid.New()
//	memberId2 := uuid.New()
//
//	test := []struct {
//		name string
//		req  *pb.GetPartyRequest
//
//		getPartyByIdReq primitive.ObjectID
//		getPartyByIdRes *model.Party
//		getPartyByIdErr error
//
//		getPartyByMemberIdReq uuid.UUID
//		getPartyByMemberIdRes *model.Party
//		getPartyByMemberIdErr error
//
//		wantRes *pb.GetPartyResponse
//		wantErr error
//	}{
//		{
//			name: "success_by_party_id",
//			req: &pb.GetPartyRequest{
//				Id: &pb.GetPartyRequest_PartyId{PartyId: partyId.Hex()},
//			},
//
//			getPartyByIdReq: partyId,
//			getPartyByIdRes: &model.Party{
//				ID:       partyId,
//				LeaderId: memberId,
//				Members:  []*model.PartyMember{{PlayerID: memberId, Username: "t"}, {PlayerID: memberId2, Username: "t2"}},
//			},
//
//			wantRes: &pb.GetPartyResponse{
//				Party: &pbmodel.Party{
//					Id:       partyId.Hex(),
//					LeaderId: memberId.String(),
//					Members:  []*pbmodel.PartyMember{{Id: memberId.String(), Username: "t"}, {Id: memberId2.String(), Username: "t2"}},
//				},
//			},
//		},
//		{
//			name: "success_by_member_id",
//			req: &pb.GetPartyRequest{
//				Id: &pb.GetPartyRequest_PlayerId{PlayerId: memberId.String()},
//			},
//
//			getPartyByMemberIdReq: memberId,
//			getPartyByMemberIdRes: &model.Party{
//				ID:       partyId,
//				LeaderId: memberId,
//				Members:  []*model.PartyMember{{PlayerID: memberId, Username: "t"}, {PlayerID: memberId2, Username: "t2"}},
//			},
//
//			wantRes: &pb.GetPartyResponse{
//				Party: &pbmodel.Party{
//					Id:       partyId.Hex(),
//					LeaderId: memberId.String(),
//					Members:  []*pbmodel.PartyMember{{Id: memberId.String(), Username: "t"}, {Id: memberId2.String(), Username: "t2"}},
//				},
//			},
//		},
//		{
//			name: "party_not_found_by_party_id",
//			req: &pb.GetPartyRequest{
//				Id: &pb.GetPartyRequest_PartyId{PartyId: partyId.Hex()},
//			},
//
//			getPartyByIdReq: partyId,
//			getPartyByIdErr: mongo.ErrNoDocuments,
//
//			wantErr: partyNotFoundErr,
//		},
//		{
//			name: "party_not_found_by_member_id",
//			req: &pb.GetPartyRequest{
//				Id: &pb.GetPartyRequest_PlayerId{PlayerId: memberId.String()},
//			},
//
//			getPartyByMemberIdReq: memberId,
//			getPartyByMemberIdErr: mongo.ErrNoDocuments,
//
//			wantErr: playerNotInPartyErr,
//		},
//	}
//
//	for _, tt := range test {
//		t.Run(tt.name, func(t *testing.T) {
//			ctx := context.Background()
//
//			// Create mock r
//			mockCntrl := gomock.NewController(t)
//			defer mockCntrl.Finish()
//
//			repo := repository.NewMockRepository(mockCntrl)
//
//			if tt.getPartyByIdReq != primitive.NilObjectID {
//				repo.EXPECT().GetPartyById(ctx, tt.getPartyByIdReq).Return(tt.getPartyByIdRes, tt.getPartyByIdErr)
//			}
//			if tt.getPartyByMemberIdReq != uuid.Nil {
//				repo.EXPECT().GetPartyByMemberId(ctx, tt.getPartyByMemberIdReq).Return(tt.getPartyByMemberIdRes, tt.getPartyByMemberIdErr)
//			}
//
//			notif := writer.NewMockNotifier(mockCntrl)
//
//			s := newPartyService(notif, repo)
//
//			res, err := s.GetParty(ctx, tt.req)
//			assert.Equalf(t, tt.wantErr, err, "wantErr: %+v, got: %+v", tt.wantErr, err)
//			assert.Equalf(t, tt.wantRes, res, "wantRes: %+v, got: %+v", tt.wantRes, res)
//		})
//	}
//}
//
//func TestPartyService_GetPartyInvites(t *testing.T) {
//	partyId := primitive.NewObjectID()
//	memberId := uuid.New()
//	memberId2 := uuid.New()
//
//	validPartyInvite := &model.PartyInvite{
//		ID:              primitive.NewObjectID(),
//		InviterID:       memberId,
//		InviterUsername: "t",
//		TargetID:        memberId2,
//		TargetUsername:  "t2",
//		ExpiresAt:       time.Now().Add(time.Hour),
//	}
//
//	var test = []struct {
//		name string
//		req  *pb.GetPartyInvitesRequest
//
//		getPartyIdByMemberIdReq uuid.UUID
//		getPartyIdByMemberIdRes primitive.ObjectID
//		getPartyIdByMemberIdErr error
//
//		getPartyInvitesByPartyIdReq primitive.ObjectID
//		getPartyInvitesByPartyIdRes []*model.PartyInvite
//		getPartyInvitesByPartyIdErr error
//
//		wantRes *pb.GetPartyInvitesResponse
//		wantErr error
//	}{
//		{
//			name: "success_by_party_id",
//			req: &pb.GetPartyInvitesRequest{
//				Id: &pb.GetPartyInvitesRequest_PartyId{PartyId: partyId.Hex()},
//			},
//
//			getPartyInvitesByPartyIdReq: partyId,
//			getPartyInvitesByPartyIdRes: []*model.PartyInvite{validPartyInvite},
//
//			wantRes: &pb.GetPartyInvitesResponse{
//				Invites: []*pbmodel.PartyInvite{validPartyInvite.ToProto()},
//			},
//		},
//		{
//			name: "success_by_member_id",
//			req: &pb.GetPartyInvitesRequest{
//				Id: &pb.GetPartyInvitesRequest_PlayerId{PlayerId: memberId.String()},
//			},
//
//			getPartyIdByMemberIdReq: memberId,
//			getPartyIdByMemberIdRes: partyId,
//
//			getPartyInvitesByPartyIdReq: partyId,
//			getPartyInvitesByPartyIdRes: []*model.PartyInvite{validPartyInvite},
//
//			wantRes: &pb.GetPartyInvitesResponse{
//				Invites: []*pbmodel.PartyInvite{validPartyInvite.ToProto()},
//			},
//		},
//		{
//			name: "party_not_found_by_party_id",
//			req: &pb.GetPartyInvitesRequest{
//				Id: &pb.GetPartyInvitesRequest_PartyId{PartyId: partyId.Hex()},
//			},
//
//			getPartyInvitesByPartyIdReq: partyId,
//			getPartyInvitesByPartyIdErr: mongo.ErrNoDocuments,
//
//			wantErr: partyNotFoundErr,
//		},
//		{
//			name: "party_not_found_by_member_id",
//			req: &pb.GetPartyInvitesRequest{
//				Id: &pb.GetPartyInvitesRequest_PlayerId{PlayerId: memberId.String()},
//			},
//
//			getPartyIdByMemberIdReq: memberId,
//			getPartyIdByMemberIdErr: mongo.ErrNoDocuments,
//
//			wantErr: playerNotInPartyErr,
//		},
//	}
//
//	for _, tt := range test {
//		t.Run(tt.name, func(t *testing.T) {
//			ctx := context.Background()
//
//			// Create mock r
//			mockCntrl := gomock.NewController(t)
//			defer mockCntrl.Finish()
//
//			repo := repository.NewMockRepository(mockCntrl)
//
//			if tt.getPartyIdByMemberIdReq != uuid.Nil {
//				repo.EXPECT().GetPartyIdByMemberId(ctx, tt.getPartyIdByMemberIdReq).Return(tt.getPartyIdByMemberIdRes, tt.getPartyIdByMemberIdErr)
//			}
//			if tt.getPartyInvitesByPartyIdReq != primitive.NilObjectID {
//				repo.EXPECT().GetPartyInvitesByPartyId(ctx, tt.getPartyInvitesByPartyIdReq).Return(tt.getPartyInvitesByPartyIdRes, tt.getPartyInvitesByPartyIdErr)
//			}
//
//			notif := writer.NewMockNotifier(mockCntrl)
//
//			s := newPartyService(notif, repo)
//
//			res, err := s.GetPartyInvites(ctx, tt.req)
//			assert.Equalf(t, tt.wantErr, err, "wantErr: %+v, got: %+v", tt.wantErr, err)
//			assert.Equalf(t, tt.wantRes, res, "wantRes: %+v, got: %+v", tt.wantRes, res)
//		})
//	}
//}
//
//// TODO
////func TestPartyService_InvitePlayer(t *testing.T) {
////	playerIds := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
////	playerUsernames := []string{"t", "t2", "t3"}
////
////	partyId := primitive.NewObjectID()
////
////	var test = []struct {
////		name string
////		req  *pb.InvitePlayerRequest
////
////		getPartyByMemberIdReq uuid.UUID
////		getPartyByMemberIdRes *model.Party
////		getPartyByMemberIdErr error
////
////		// Note, will only ever get called if requester != leader
////		getPartySettingsRes *model.PartySettings
////		getPartySettingsErr error
////
////		getPartyIdOfTargetReq uuid.UUID
////		getPartyIdOfTargetRes primitive.ObjectID
////		getPartyIdOfTargetErr error
////
////		doesPartyInviteExistReqPartyId  primitive.ObjectID
////		doesPartyInviteExistReqTargetId uuid.UUID
////		doesPartyInviteExistRes         bool
////		doesPartyInviteExistErr         error
////
////		createPartyInviteReq *model.PartyInvite
////		createPartyInviteErr error
////
////		wantRes *pb.InvitePlayerResponse
////		wantErr error
////	}{
////		{
////			name: "success",
////			req: &pb.InvitePlayerRequest{
////				IssuerId:       playerIds[0].String(),
////				IssuerUsername: playerUsernames[0],
////
////				TargetID:       playerIds[0].String(),
////				TargetUsername: playerUsernames[0],
////			},
////
////			getPartyByMemberIdReq: playerIds[0],
////			getPartyByMemberIdRes: &model.Party{
////				PlayerID:       partyId,
////				LeaderId: playerIds[0],
////				Members:  []*model.PartyMember{{PlayerID: playerIds[0], Username: playerUsernames[0]}},
////			},
////
////			getPartyIdOfTargetReq: playerIds[1],
////			getPartyIdOfTargetRes: primitive.NilObjectID,
////
////			doesPartyInviteExistReqPartyId:  partyId,
////			doesPartyInviteExistReqTargetId: playerIds[1],
////			doesPartyInviteExistRes:         false,
////
////			createPartyInviteReq: &model.PartyInvite{
////				PartyID:  partyId,
////			}
////		},
////		// Player 0 is inviting player 1 to the party. Player 2 is the leader and the party has all invites enabled.
////		{
////			name: "success_member_inviting",
////			req: &pb.InvitePlayerRequest{
////				IssuerId:       playerIds[0].String(),
////				IssuerUsername: playerUsernames[0],
////
////				TargetID:       playerIds[1].String(),
////				TargetUsername: playerUsernames[1],
////			},
////
////			getPartyByMemberIdReq: playerIds[0],
////			getPartyByMemberIdRes: &model.Party{
////				PlayerID:       partyId,
////				LeaderId: playerIds[2],
////				Members:  []*model.PartyMember{{PlayerID: playerIds[0], Username: playerUsernames[0]}, {PlayerID: playerIds[2], Username: playerUsernames[2]}},
////			},
////
////			getPartySettingsReq: partyId,
////			getPartySettingsRes: &model.PartySettings{
////				AllowMemberInvite: true,
////			},
////
////			getPartyIdOfTargetReq: playerIds[1],
////			getPartyIdOfTargetRes: primitive.NilObjectID,
////
////			doesPartyInviteExistReq: &model.PartyInvite{},
////		},
////	}
////}
