package api

import (
	"github.com/kevinburke/nacl"
	"bytes"
	"net/http"
	"io/ioutil"
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