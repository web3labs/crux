package main

import (
	"log"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"github.com/blk-io/chimera-api/chimera"
)
func main() {
	var conn *grpc.ClientConn
	// Initiate a connection with the server
	conn, err := grpc.Dial("passthrough:///unix://qdata", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}
	defer conn.Close()
	c := chimera.NewClientClient(conn)
	response, err := c.Upcheck(context.Background(), &chimera.UpCheckResponse{Message: "foo"})
	if err != nil {
		log.Fatalf("error when calling Upcheck: %s", err)
	}
	log.Printf("Response from server: %s", response.Message)
}
