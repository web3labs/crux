package api

import (
	"bytes"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"
	log "github.com/sirupsen/logrus"
	"github.com/kevinburke/nacl"
)

type EncryptedPayload struct {
	Sender         nacl.Key
	CipherText     []byte
	Nonce          nacl.Nonce
	RecipientBoxes [][]byte
	RecipientNonce nacl.Nonce
}

type PartyInfo struct {
	url string
	// public key -> URL
	recipients map[string]string
	parties    map[string]bool // URLs
}

func (s *PartyInfo) GetRecipient(key string) (string, bool) {
	value, ok := s.recipients[key]
	return value, ok
}

func LoadPartyInfo(url string, otherNodes []string) PartyInfo {
	parties := make(map[string]bool)
	for _, node := range otherNodes {
		parties[node] = true
	}

	return PartyInfo{
		url:        url,
		recipients: make(map[string]string),
		parties:    parties,
	}
}

func (s *PartyInfo) GetPartyInfo() {
	encodedPartyInfo := EncodePartyInfo(*s)

	// First copy our endpoints as we update this map in place
	urls := make(map[string]bool)
	for k, v := range s.parties {
		urls[k] = v
	}

	for url := range urls {
		resp, err := http.Post(
			url + "/partyinfo", "application/octet-stream", bytes.NewReader(encodedPartyInfo))
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
		s.UpdatePartyInfo(encoded)
	}
}

func (s *PartyInfo) PollPartyInfo() {
	time.Sleep(time.Duration(rand.Intn(16)) * time.Second)
	s.GetPartyInfo()

	ticker := time.NewTicker(2 * time.Minute)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <- ticker.C:
				s.GetPartyInfo()
			case <- quit:
				ticker.Stop()
				return
			}
		}
	}()
}

// This can happen from the /partyinfo server endpoint being hit, or
// by a response from us hitting another nodes /partyinfo endpoint
// TODO: Control access via a channel for updates
func (s *PartyInfo) UpdatePartyInfo(encoded []byte) {
	pi := DecodePartyInfo(encoded)

	for publicKey, url := range pi.recipients {
		// we should ignore messages about ourselves
		// in order to stop people masquerading as you, there
		// should be a digital signature associated with each
		// url -> node broadcast
		if url != s.url {
			s.recipients[publicKey] = url
		}
	}

	for url := range pi.parties {
		// we don't want to broadcast party info to ourselves
		if url != s.url {
			s.parties[url] = true
		}
	}
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
