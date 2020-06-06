package server

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/blk-io/chimera-api/chimera"
	"github.com/blk-io/crux/api"
	"github.com/kevinburke/nacl"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

type Server struct {
	Enclave Enclave
}

func (s *Server) Version(ctx context.Context, in *chimera.ApiVersion) (*chimera.ApiVersion, error) {
	return &chimera.ApiVersion{Version: apiVersion}, nil
}

func (s *Server) Upcheck(ctx context.Context, in *chimera.UpCheckResponse) (*chimera.UpCheckResponse, error) {
	return &chimera.UpCheckResponse{Message: upCheckResponse}, nil
}
func (s *Server) Send(ctx context.Context, in *chimera.SendRequest) (*chimera.SendResponse, error) {
	key, err := s.processSend(in.GetFrom(), in.GetTo(), &in.Payload)
	var sendResp chimera.SendResponse
	if err != nil {
		log.Error(err)
	} else {
		sendResp = chimera.SendResponse{Key: key}
	}
	return &sendResp, err
}

func (s *Server) processSend(b64from string, b64recipients []string, payload *[]byte) ([]byte, error) {
	log.WithFields(log.Fields{
		"b64From":       b64from,
		"b64Recipients": b64recipients,
		"payload":       hex.EncodeToString(*payload)}).Debugf(
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

func (s *Server) Receive(ctx context.Context, in *chimera.ReceiveRequest) (*chimera.ReceiveResponse, error) {
	payload, err := s.processReceive(in.Key, in.To)
	var receiveResp chimera.ReceiveResponse
	if err != nil {
		log.Error(err)
	} else {
		receiveResp = chimera.ReceiveResponse{Payload: payload}
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

func (s *Server) UpdatePartyInfo(ctx context.Context, in *chimera.PartyInfo) (*chimera.PartyInfoResponse, error) {
	recipients := make(map[[nacl.KeySize]byte]string)
	for url, key := range in.Recipients {
		var as [32]byte
		copy(as[:], key)
		recipients[as] = url
	}
	s.Enclave.UpdatePartyInfoGrpc(in.Url, recipients, in.Parties)
	encoded := s.Enclave.GetEncodedPartyInfoGrpc()
	var decodedPartyInfo chimera.PartyInfoResponse
	err := json.Unmarshal(encoded, &decodedPartyInfo)
	if err != nil {
		log.Errorf("Unmarshalling failed with %v", err)
	}
	return &chimera.PartyInfoResponse{Payload: decodedPartyInfo.Payload}, nil
}

func (s *Server) Push(ctx context.Context, in *chimera.PushPayload) (*chimera.PartyInfoResponse, error) {
	sender := new([nacl.KeySize]byte)
	nonce := new([nacl.NonceSize]byte)
	recipientNonce := new([nacl.NonceSize]byte)
	copy((*sender)[:], in.Ep.Sender)
	copy((*nonce)[:], in.Ep.Nonce)
	copy((*recipientNonce)[:], in.Ep.ReciepientNonce)

	encyptedPayload := api.EncryptedPayload{
		Sender:         sender,
		CipherText:     in.Ep.CipherText,
		Nonce:          nonce,
		RecipientBoxes: in.Ep.ReciepientBoxes,
		RecipientNonce: recipientNonce,
	}

	digestHash, err := s.Enclave.StorePayloadGrpc(encyptedPayload, in.Encoded)
	if err != nil {
		log.Fatalf("Unable to store payload, error: %s\n", err)
	}

	return &chimera.PartyInfoResponse{Payload: digestHash}, nil
}

func (s *Server) Delete(ctx context.Context, in *chimera.DeleteRequest) (*chimera.DeleteRequest, error) {
	var deleteReq chimera.DeleteRequest
	err := s.Enclave.Delete(&deleteReq.Key)
	if err != nil {
		log.Fatalf("Unable to delete payload, error: %s\n", err)
	}
	return &chimera.DeleteRequest{Key: deleteReq.Key}, nil
}

func (s *Server) Resend(ctx context.Context, in *chimera.ResendRequest) (*chimera.ResendResponse, error) {
	var resendReq chimera.ResendRequest
	var err error

	if resendReq.Type == "all" {
		err = s.Enclave.RetrieveAllFor(&resendReq.PublicKey)
		if err != nil {
			log.Fatalf("Invalid body, exited with %s", err)
		}
		return nil, err
	} else if resendReq.Type == "individual" {
		var encodedPl *[]byte
		encodedPl, err = s.Enclave.RetrieveFor(&resendReq.Key, &resendReq.PublicKey)
		if err != nil {
			log.Fatalf("Invalid body, exited with %s", err)
			return nil, err
		}
		return &chimera.ResendResponse{Encoded: *encodedPl}, nil
	}
	return nil, err
}

func decodeErrorGRPC(name string, value string, err error) {
	log.Error(fmt.Sprintf("Invalid request: unable to decode %s: %s, error: %s\n",
		name, value, err))
}
