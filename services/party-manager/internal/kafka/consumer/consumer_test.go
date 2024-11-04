package consumer

//
//import (
//	"github.com/emortalmc/mono-services/services/party-manager/internal/repository/model"
//	"github.com/emortalmc/mono-services/services/party-manager/internal/utils"
//	"testing"
//)
//
//var testUuids = utils.RandomUuidSlice(15)
//
//func createPartyWithMemberCount(mCount int) *model.Party {
//	members := make([]*model.PartyMember, mCount)
//	for i := 0; i < mCount; i++ {
//		members[i] = &model.PartyMember{PlayerID: testUuids[i]}
//	}
//
//	return &model.Party{Members: members, LeaderId: testUuids[0]}
//}
//
//func TestElectNewPartyLeader(t *testing.T) {
//	partyMap := make(map[int]*model.Party)
//	for i := 1; i <= 10; i++ { // parties with 1-10 members
//		partyMap[i] = createPartyWithMemberCount(i)
//	}
//
//	tests := []struct {
//		name    string
//		repeats int
//
//		party *model.Party
//
//		acceptableResults []*model.PartyMember
//		wantErr           error
//	}{
//		{
//			name:    "one member",
//			party:   partyMap[1],
//			wantErr: leaderElectionNotEnoughMembersErr,
//		},
//		{
//			name:  "two members",
//			party: partyMap[2],
//		},
//		{
//			name:    "three members",
//			repeats: 300,
//			party:   partyMap[3],
//		},
//		{
//			name:    "four members",
//			repeats: 400,
//			party:   partyMap[4],
//		},
//		{
//			name:    "10 members",
//			repeats: 750,
//			party:   partyMap[10],
//		},
//	}
//
//	for _, test := range tests {
//		for i := 0; i < test.repeats; i++ {
//			t.Run(test.name, func(t *testing.T) {
//				res, err := electNewPartyLeader(test.party)
//				if err != test.wantErr {
//					t.Errorf("got err %v, want %v", err, test.wantErr)
//				}
//
//				if test.wantErr != nil {
//					return
//				}
//
//				for _, acceptableResult := range test.party.Members[1:] {
//					if res == acceptableResult {
//						return
//					}
//				}
//
//				t.Errorf("got %v, want one of %v", res, test.acceptableResults)
//			})
//		}
//	}
//}
