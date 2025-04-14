package main

import (
	"context"
	pb "github.com/emortalmc/proto-specs/gen/go/grpc/party"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"time"
)

func main() {
	conn, err := grpc.NewClient("localhost:10006", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}

	partyClient := pb.NewPartyServiceClient(conn)

	startTime := time.Now().UnixMilli()
	res, err := partyClient.EmptyParty(context.Background(), &pb.EmptyPartyRequest{
		Id: &pb.EmptyPartyRequest_PlayerId{PlayerId: "8d36737e-1c0a-4a71-87de-9906f577845e"},
	})
	if err != nil {
		panic(err)
	}

	log.Printf("response: %+v", res)
	log.Printf("took: %dms", time.Now().UnixMilli()-startTime)
}
