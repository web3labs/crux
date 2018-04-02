package api

import (
	"bytes"
	"net/http"
	"io/ioutil"
	"github.com/kevinburke/nacl"
	log "github.com/sirupsen/logrus"
	"gitlab.com/blk-io/crux/server"
	"time"
	"math/rand"
)

type EncryptedPayload struct {
	Sender         nacl.Key
	CipherText     []byte
	Nonce          nacl.Nonce
	RecipientBoxes [][]byte
	RecipientNonce nacl.Nonce
}

type PartyInfo struct {
	Url string
	// public key -> URL
	Recipients map[string]string
	Parties map[string]bool  // URLs
}

func LoadPartyInfo(url string, otherNodes []string) PartyInfo {
	parties := make(map[string]bool)
	for _, node := range otherNodes {
		parties[node] = true
	}

	return PartyInfo{
		Url: url,
		Recipients: make(map[string]string),
		Parties: parties,
	}
}

func (s *PartyInfo) GetPartyInfo(tm server.TransactionManager) {
	partyInfo := tm.Enclave.GetEncodedPartyInfo()

	// First copy our endpoints as we update this map in place
	urls := make(map[string]bool)
	for k, v := range s.Parties {
		urls[k] = v
	}

	for url := range urls {
		resp, err := http.Post(
			url + "/partyinfo", "application/octet-stream", bytes.NewReader(partyInfo))
		if err != nil {
			log.WithField("url", url).Errorf(
				"Error sending /partyinfo request, %v", err)
			break
		}

		var encoded []byte
		encoded, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			log.WithField("url", url).Errorf(
				"Unable to read partInfo response from host, %v", err)
			break
		}
		tm.Enclave.UpdatePartyInfo(encoded)
	}
}

func (s *PartyInfo) PollPartyInfo(tm server.TransactionManager) {
	time.Sleep(time.Duration(rand.Intn(16)) * time.Second)
	s.GetPartyInfo(tm)

	ticker := time.NewTicker(2 * time.Minute)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <- ticker.C:
				s.GetPartyInfo(tm)
			case <- quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func Push(encoded []byte, url string) (string, error) {

	resp, err := http.Post(
		url + "/push", "application/octet-stream", bytes.NewReader(encoded))
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		return "", err
	}

	return string(body), nil
}
