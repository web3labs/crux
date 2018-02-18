package server

import (
	"encoding/json"
	"encoding/base64"
	"fmt"
	"net/http"
	"github.com/blk-io/crux/enclave"
	"github.com/kevinburke/nacl"
	"github.com/blk-io/crux/api"
)

type TransactionManager struct {
	Key nacl.Key
	Enclave enclave.Enclave
}

func (s *TransactionManager) Upcheck(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "I'm up!")
}

func (s *TransactionManager) Send(w http.ResponseWriter, req *http.Request) {
	var sendReq api.SendRequest
	if err := json.NewDecoder(req.Body).Decode(&sendReq); err != nil {
		req.Body.Close()
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid request: %s\n", req.URL)
	} else {
		payload, err := base64.StdEncoding.DecodeString(sendReq.Payload)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Invalid request: %s, unable to decode payload: %s\n",
				req.URL, sendReq.Payload)
		}

		key := s.Enclave.Store(s.Key, &payload)
		encodedKey := base64.StdEncoding.EncodeToString(key)
		sendResp := api.SendResponse{Key : encodedKey}
		json.NewEncoder(w).Encode(sendResp)
		w.Header().Set("Content-Type", "application/json")
	}
}

func (s *TransactionManager) Receive(w http.ResponseWriter, req *http.Request) {
	var receiveReq api.ReceiveRequest
	if err := json.NewDecoder(req.Body).Decode(&receiveReq); err != nil {
		req.Body.Close()
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid request: %s\n", req.URL)
	} else {
		key, err := base64.StdEncoding.DecodeString(receiveReq.Key)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Invalid request: %s, unable to decode payload: %s\n",
				req.URL, receiveReq.Key)
		}

		payload := s.Enclave.Retrieve(s.Key, &key)
		encodedPayload := base64.StdEncoding.EncodeToString(payload)
		sendResp := api.ReceiveResponse{Payload : encodedPayload}
		json.NewEncoder(w).Encode(sendResp)
		w.Header().Set("Content-Type", "application/json")
	}
}

func (s *TransactionManager) Delete(w http.ResponseWriter, req *http.Request) {
	var deleteReq api.DeleteRequest
	if err := json.NewDecoder(req.Body).Decode(&deleteReq); err != nil {
		req.Body.Close()
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid request: %s\n", req.URL)
	} else {
		key, err := base64.StdEncoding.DecodeString(deleteReq.Key)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Invalid request: %s, unable to decode payload: %s\n",
				req.URL, deleteReq.Key)
		}

		s.Enclave.Delete(&key)
	}
}