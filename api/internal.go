package api

import (
	"bytes"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"
	log "github.com/sirupsen/logrus"
	"github.com/kevinburke/nacl"
	"gitlab.com/blk-io/crux/utils"
	"encoding/hex"
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
	recipients map[[nacl.KeySize]byte]string
	parties    map[string]bool // URLs
	client     utils.HttpClient
}

func (s *PartyInfo) GetRecipient(key nacl.Key) (string, bool) {
	value, ok := s.recipients[*key]
	return value, ok
}

func InitPartyInfo(rawUrl string, otherNodes []string, client utils.HttpClient) PartyInfo {
	parties := make(map[string]bool)
	for _, node := range otherNodes {
		parties[node] = true
	}

	return PartyInfo{
		url:        rawUrl,
		recipients: make(map[[nacl.KeySize]byte]string),
		parties:    parties,
		client:     client,
	}
}

func CreatePartyInfo(
	url string,
	otherNodes []string,
	otherKeys []nacl.Key,
	client utils.HttpClient) PartyInfo {

	recipients := make(map[[nacl.KeySize]byte]string)
	parties := make(map[string]bool)
	for i, node := range otherNodes {
		parties[node] = true
		recipients[*otherKeys[i]] = node
	}

	return PartyInfo{
		url:        url,
		recipients: recipients,
		parties:    parties,
		client:     client,
	}
}

func (s *PartyInfo) RegisterPublicKeys(pubKeys []nacl.Key) {
	for _, pubKey := range pubKeys {
		s.recipients[*pubKey] = s.url
	}
}

func (s *PartyInfo) GetPartyInfo() {
	encodedPartyInfo := EncodePartyInfo(*s)

	// First copy our endpoints as we update this map in place
	urls := make(map[string]bool)
	for k, v := range s.parties {
		urls[k] = v
	}

	for rawUrl := range urls {
		if rawUrl == s.url {
			continue
		}

		endPoint, err := utils.BuildUrl(rawUrl, "/partyinfo")

		if err != nil {
			log.WithFields(log.Fields{"rawUrl": rawUrl, "endPoint": "/partyinfo"}).Errorf(
				"Invalid endpoint provided")
		}

		var req *http.Request
		req, err = http.NewRequest("POST", endPoint, bytes.NewBuffer(encodedPartyInfo[:]))

		if err != nil {
			log.WithField("url", rawUrl).Errorf(
				"Error creating /partyinfo request, %v", err)
			break
		}
		req.Header.Set("Content-Type", "application/octet-stream")

		log.Debugf("%s %s %s", req.RemoteAddr, req.Method, req.URL)
		resp, err := s.client.Do(req)
		if err != nil {
			log.WithField("url", rawUrl).Errorf(
				"Error sending /partyinfo request, %v", err)
			continue
		}

		var encoded []byte
		encoded, err = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.WithField("url", rawUrl).Errorf(
				"Unable to read partyInfo response from host, %v", err)
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
	log.Debugf("Updating party info payload: %s", hex.EncodeToString(encoded))
	pi, err := DecodePartyInfo(encoded)

	if err != nil {
		log.WithField("encoded", encoded).Errorf(
			"Unable to decode party info, error: %v", err)
	}

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
		s.parties[url] = true
	}
}

func Push(encoded []byte, url string, client utils.HttpClient) (string, error) {

	endPoint, err := utils.BuildUrl(url, "/push")
	if err != nil {
		return "", err
	}

	var req *http.Request
	req, err = http.NewRequest("POST", endPoint, bytes.NewReader(encoded))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := client.Do(req)
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
