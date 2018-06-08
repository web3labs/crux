package server

import (
	"golang.org/x/net/context"
	log "github.com/sirupsen/logrus"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

type Server struct {
	Enclave Enclave
}

func (s *Server) Version(ctx context.Context, in *ApiVersion) (*ApiVersion, error) {
	return &ApiVersion{Version:apiVersion}, nil
}

func (s *Server) Upcheck(ctx context.Context, in *UpCheckResponse) (*UpCheckResponse, error) {
	return &UpCheckResponse{Message:upCheckResponse}, nil
}
func (s *Server) Send(ctx context.Context, in *SendRequest) (*SendResponse, error) {
	key, err := s.processSend(in.GetFrom(), in.GetTo(), &in.Payload)
	var sendResp SendResponse
	if err != nil {
		log.Error(err)
	} else {
		sendResp = SendResponse{Key: key}
	}
	return &sendResp, err
}

func (s *Server) processSend(b64from string, b64recipients []string, payload *[]byte) ([]byte, error) {
	log.WithFields(log.Fields{
		"b64From" : b64from,
		"b64Recipients": b64recipients,
		"payload": hex.EncodeToString(*payload),}).Debugf(
		"Processing send request")

	sender, err := base64.StdEncoding.DecodeString(b64from)
	if err != nil {
		decodeErrorGRPC("sender", b64from, err)
		return nil, err
	}

	recipients := make([][]byte, len(b64recipients))
	for i, value := range b64recipients {
		recipient, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			decodeErrorGRPC("recipients", value, err)
			return nil, err
		} else {
			recipients[i] = recipient
		}
	}

	return s.Enclave.Store(payload, sender, recipients)
}

func (s *Server) Receive(ctx context.Context, in *ReceiveRequest) (*ReceiveResponse, error) {
	payload, err := s.processReceive(in.Key, in.To)
	var receiveResp ReceiveResponse
	if err != nil {
		log.Error(err)
	} else {
		receiveResp = ReceiveResponse{Payload: payload}
	}
	return &receiveResp, err
}

func (s *Server) processReceive(b64Key []byte, b64To string) ([]byte, error) {
	if b64To != "" {
		to, err := base64.StdEncoding.DecodeString(b64To)
		if err != nil {
			return nil, fmt.Errorf("unable to decode to: %s", b64Key)
		}

		return s.Enclave.Retrieve(&b64Key, &to)
	} else {
		return s.Enclave.RetrieveDefault(&b64Key)
	}
}

func decodeErrorGRPC(name string, value string, err error) {
	log.Error(fmt.Sprintf("Invalid request: unable to decode %s: %s, error: %s\n",
		name, value, err))
}
