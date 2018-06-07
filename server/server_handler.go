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

func (s *Server) Version(ctx context.Context, in *Tm) (*ApiVersion, error) {
	return &ApiVersion{Version:apiVersion}, nil
}

func (s *Server) Upcheck(ctx context.Context, in *Tm) (*UpCheckResponse, error) {
	return &UpCheckResponse{UpCheck:upCheckResponse}, nil
}
func (s *Server) Send(ctx context.Context, in *SendRequest) (*SendResponse, error) {
	payload, err := base64.StdEncoding.DecodeString(in.Payload)
	if err != nil {
		decodeErrorGRPC("payload", in.GetPayload(), err)
	}

	key, err := s.processSend(in.GetFrom(), in.GetTo(), &payload)
	var sendResp SendResponse
	if err != nil {
		log.Error(err)
	} else {
		encodedKey := base64.StdEncoding.EncodeToString(key)
		sendResp = SendResponse{Key: encodedKey}
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

func decodeErrorGRPC(name string, value string, err error) {
	log.Error(fmt.Sprintf("Invalid request: unable to decode %s: %s, error: %s\n",
		name, value, err))
}
