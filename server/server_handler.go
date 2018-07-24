package server

import (
	"golang.org/x/net/context"
	log "github.com/sirupsen/logrus"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/kevinburke/nacl"
	"encoding/json"
	"github.com/blk-io/crux/api"
	"github.com/blk-io/chimera-api/protofiles"
)

type Server struct {
	Enclave Enclave
}

func (s *Server) Version(ctx context.Context, in *protofiles.ApiVersion) (*protofiles.ApiVersion, error) {
	return &protofiles.ApiVersion{Version:apiVersion}, nil
}

func (s *Server) Upcheck(ctx context.Context, in *protofiles.UpCheckResponse) (*protofiles.UpCheckResponse, error) {
	return &protofiles.UpCheckResponse{Message:upCheckResponse}, nil
}
func (s *Server) Send(ctx context.Context, in *protofiles.SendRequest) (*protofiles.SendResponse, error) {
	key, err := s.processSend(in.GetFrom(), in.GetTo(), &in.Payload)
	var sendResp protofiles.SendResponse
	if err != nil {
		log.Error(err)
	} else {
		sendResp = protofiles.SendResponse{Key: key}
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

func (s *Server) Receive(ctx context.Context, in *protofiles.ReceiveRequest) (*protofiles.ReceiveResponse, error) {
	payload, err := s.processReceive(in.Key, in.To)
	var receiveResp protofiles.ReceiveResponse
	if err != nil {
		log.Error(err)
	} else {
		receiveResp = protofiles.ReceiveResponse{Payload: payload}
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

func (s *Server) UpdatePartyInfo(ctx context.Context, in *protofiles.PartyInfo) (*protofiles.PartyInfoResponse, error) {
	recipients := make(map[[nacl.KeySize]byte]string)
	for url, key := range in.Recipients{
		var as [32]byte
		copy(as[:], key)
		recipients[as] = url
	}
	s.Enclave.UpdatePartyInfoGrpc(in.Url, recipients, in.Parties)
	encoded := s.Enclave.GetEncodedPartyInfoGrpc()
	var decodedPartyInfo protofiles.PartyInfoResponse
	err := json.Unmarshal(encoded, &decodedPartyInfo)
	if err != nil{
		log.Errorf("Unmarshalling failed with %v", err)
	}
	return &protofiles.PartyInfoResponse{Payload: decodedPartyInfo.Payload}, nil
}


func (s *Server) Push(ctx context.Context, in *protofiles.PushPayload) (*protofiles.PartyInfoResponse, error) {
	sender := new([nacl.KeySize]byte)
	nonce := new([nacl.NonceSize]byte)
	recipientNonce := new([nacl.NonceSize]byte)
	copy((*sender)[:], in.Ep.Sender)
	copy((*nonce)[:], in.Ep.Nonce)
	copy((*recipientNonce)[:], in.Ep.ReciepientNonce)

	encyptedPayload := api.EncryptedPayload{
		Sender:sender,
		CipherText:in.Ep.CipherText,
		Nonce:nonce,
		RecipientBoxes:in.Ep.ReciepientBoxes,
		RecipientNonce:recipientNonce,
	}

	digestHash, err := s.Enclave.StorePayloadGrpc(encyptedPayload, in.Encoded)
	if err != nil {
		log.Fatalf("Unable to store payload, error: %s\n", err)
	}

	return &protofiles.PartyInfoResponse{Payload: digestHash}, nil
}

func decodeErrorGRPC(name string, value string, err error) {
	log.Error(fmt.Sprintf("Invalid request: unable to decode %s: %s, error: %s\n",
		name, value, err))
}
