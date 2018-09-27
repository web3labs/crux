package client

import (
	"encoding/base64"
	"github.com/blk-io/chimera-api/chimera"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"reflect"
	"testing"
)

const sender = "zSifTnkv5r4K67Dq304eVcM4FpxGfHLe1yTCBm0/7wg="
const receiver = "I/EbshW61ykJ+qTivXPaKyQ5WmQDUFfMNGEBj2E2uUs="

var payload = []byte("payload")

func TestIntegration(t *testing.T) {
	var conn1 *grpc.ClientConn
	var conn2 *grpc.ClientConn
	// Initiate a connection with the first server
	conn1, err := grpc.Dial(":9020", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("did not connect: %s", err)
	}
	defer conn1.Close()
	c1 := chimera.NewClientClient(conn1)
	// Initiate a connection with the second server
	conn2, err = grpc.Dial(":9025", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("did not connect: %s", err)
	}
	defer conn2.Close()
	c2 := chimera.NewClientClient(conn2)

	Upcheckresponse1, err := c1.Upcheck(context.Background(), &chimera.UpCheckResponse{Message: "foo"})
	if err != nil {
		t.Fatalf("error when calling Upcheck: %s", err)
	}
	t.Logf("Response from server: %s", Upcheckresponse1.Message)

	Upcheckresponse2, err := c2.Upcheck(context.Background(), &chimera.UpCheckResponse{Message: "foo"})
	if err != nil {
		t.Fatalf("error when calling Upcheck: %s", err)
	}
	t.Logf("Response from server: %s", Upcheckresponse2.Message)

	sendReqs := []chimera.SendRequest{
		{
			Payload: []byte("payload"),
			From:    sender,
			To:      []string{receiver},
		},
		{
			Payload: []byte("test"),
			To:      []string{},
		},
		{
			Payload: []byte("blk-io crux"),
		},
	}

	sendResponse := chimera.SendResponse{}
	for _, sendReq := range sendReqs {
		sendResp, err := c1.Send(context.Background(), &sendReq)
		if err != nil {
			t.Fatalf("gRPC send failed with %s", err)
		}
		sendResponse = chimera.SendResponse{Key: sendResp.Key}
		t.Logf("The response for Send request is: %s", base64.StdEncoding.EncodeToString(sendResponse.Key))

		recResp, err := c1.Receive(context.Background(), &chimera.ReceiveRequest{Key: sendResponse.Key, To: receiver})
		if err != nil {
			t.Fatalf("gRPC receive failed with %s", err)
		}
		receiveResponse := chimera.ReceiveResponse{Payload: recResp.Payload}
		if !reflect.DeepEqual(receiveResponse.Payload, sendReq.Payload) {
			t.Fatalf("handler returned unexpected response: %v, expected: %v\n", receiveResponse.Payload, sendReq.Payload)
		} else {
			t.Logf("The payload return is %v", receiveResponse.Payload)
		}
	}
}
