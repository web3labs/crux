package api

import (
	"bytes"
	"io/ioutil"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"net/http"
	"time"
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

func (s *PartyInfo) GetPartyInfo() {
	encodedPartyInfo := EncodePartyInfo(*s)

	// First copy our endpoints as we update this map in place
	urls := make(map[string]bool)
	for k, v := range s.Parties {
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

func (s *PartyInfo) UpdatePartyInfo(encoded []byte) {
	pi := DecodePartyInfo(encoded)

	for publicKey, url := range pi.Recipients {
		// we should ignore messages about ourselves
		// in order to stop people masquerading as you, there
		// should be a digital signature associated with each
		// url -> node broadcast
		if url != s.Url {
			s.Recipients[publicKey] = url
		}
	}

	for url := range pi.Parties {
		// we don't want to broadcast party info to ourselves
		if url != s.Url {
			s.Parties[url] = true
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
