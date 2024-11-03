package gsallocation

import (
	allocv1 "agones.dev/agones/pkg/apis/allocation/v1"
	v1 "agones.dev/agones/pkg/client/clientset/versioned/typed/allocation/v1"
	"context"
	"fmt"
	"github.com/emortalmc/mono-services/services/matchmaker/internal/utils"
	pb "github.com/emortalmc/proto-specs/gen/go/model/matchmaker"
	kubev1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"strconv"
	"sync"
)

func AllocateServers(ctx context.Context, allocationClient v1.GameServerAllocationInterface,
	allocations map[*pb.Match]*allocv1.GameServerAllocation) map[*pb.Match]error {

	errors := make(map[*pb.Match]error)
	errorsLock := sync.Mutex{}

	wg := sync.WaitGroup{}
	wg.Add(len(allocations))

	for match, allocation := range allocations {
		go func(fMatch *pb.Match, fAllocation *allocv1.GameServerAllocation) {
			defer wg.Done()
			if err := AllocateServer(ctx, allocationClient, fMatch, fAllocation); err != nil {
				errorsLock.Lock()
				defer errorsLock.Unlock()
				errors[fMatch] = err
			}
		}(match, allocation)
	}

	wg.Wait()
	return errors
}

func AllocateServer(ctx context.Context, allocationClient v1.GameServerAllocationInterface,
	match *pb.Match, allocationReq *allocv1.GameServerAllocation) error {

	resp, err := allocationClient.Create(ctx, allocationReq, kubev1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create allocation: %w", err)
	}

	allocation := resp.Status
	if allocation.State != allocv1.GameServerAllocationAllocated {
		return fmt.Errorf("allocation was not successful: %s", allocation.State)
	}

	log.Printf("Allocated: %+v", allocation)
	log.Printf("Metadata: %+v", allocation.Metadata)

	protocolVersion, versionName := parseVersions(allocation.Metadata.Annotations)

	match.Assignment = &pb.Assignment{
		ServerId:        allocation.GameServerName,
		ServerAddress:   allocation.Address,
		ServerPort:      uint32(allocation.Ports[0].Port),
		ProtocolVersion: protocolVersion,
		VersionName:     versionName,
	}

	return err
}

func parseVersions(annotations map[string]string) (protocolVersion *int64, versionName *string) {
	protocolStr, ok := annotations["agones.dev/sdk-emc-protocol-version"]
	if ok {
		protocol, err := strconv.Atoi(protocolStr)
		if err == nil {
			protocolVersion = utils.PointerOf(int64(protocol))
		}
	}

	version, ok := annotations["agones.dev/sdk-emc-version-name"]
	if ok {
		versionName = &version
	}

	return protocolVersion, versionName
}
