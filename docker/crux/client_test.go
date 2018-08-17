package client

import (
		"golang.org/x/net/context"
	"google.golang.org/grpc"
	"github.com/blk-io/chimera-api/chimera"
	"encoding/base64"
	"reflect"
	"testing"
)

const sender = "BULeR8JyUWhiuuCMU/HLA0Q5pzkYT+cHII3ZKBey3Bo="
const receiver = "QfeDAys9MPDs2XHExtc84jKGHxZg/aj52DTh0vtA3Xc="

var payload = []byte("payload")

func TestIntegration(t *testing.T) {
	var conn *grpc.ClientConn
	// Initiate a connection with the server
	conn, err := grpc.Dial("passthrough:///unix:///go/src/crux/crux.ipc", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("did not connect: %s", err)
	}
	defer conn.Close()
	c := chimera.NewClientClient(conn)
	Upcheckresponse, err := c.Upcheck(context.Background(), &chimera.UpCheckResponse{Message: "foo"})
	if err != nil {
		t.Fatalf("error when calling Upcheck: %s", err)
	}
	t.Logf("Response from server: %s", Upcheckresponse.Message)

	sendReqs := []chimera.SendRequest{
		{
			Payload: []byte("payload"),
			From: sender,
			To: []string{receiver},
		},
		{
			Payload: []byte("test"),
			To: []string{},
		},
		{
			Payload: []byte("blk-io crux"),
		},
	}

	sendResponse := chimera.SendResponse{}
	for _, sendReq := range sendReqs {
		sendResp, err:= c.Send(context.Background(), &sendReq)
		if err != nil {
			t.Fatalf("gRPC send failed with %s", err)
		}
		sendResponse = chimera.SendResponse{Key:sendResp.Key}
		t.Logf("The response for Send request is: %s", base64.StdEncoding.EncodeToString(sendResponse.Key))

		recResp, err:= c.Receive(context.Background(), &chimera.ReceiveRequest{Key:sendResponse.Key, To:receiver})
		if err != nil {
			t.Fatalf("gRPC receive failed with %s", err)
		}
		receiveResponse := chimera.ReceiveResponse{Payload:recResp.Payload}
		if !reflect.DeepEqual(receiveResponse.Payload, sendReq.Payload) {
			t.Fatalf("handler returned unexpected response: %v, expected: %v\n", receiveResponse.Payload, sendReq.Payload)
		} else {
			t.Logf("The payload return is %v", receiveResponse.Payload)
		}
	}
}
