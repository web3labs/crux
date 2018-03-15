package enclave

import (
	"github.com/kevinburke/nacl"
	"github.com/kevinburke/nacl/box"
	"github.com/kevinburke/nacl/secretbox"
	"github.com/blk-io/crux/storage"
	"golang.org/x/crypto/sha3"
	"github.com/blk-io/crux/api"
	log "github.com/sirupsen/logrus"
	"net/http"
	"bytes"
	"io/ioutil"
)

type Enclave struct {
	Db       storage.DataStore
	pubKeys  []nacl.Key
	privKeys []nacl.Key
	partyInfo api.PartyInfo
}

func (s *Enclave) Store(
	message *[]byte, sender nacl.Key, recipients []string) ([]byte, error) {

	// Check we have private key of requested sender public key
	nonce := nacl.NewNonce()
	masterKey := nacl.NewKey()
	recipientNonce := nacl.NewNonce()

	sealedMessage := secretbox.Seal([]byte{}, *message, nonce, masterKey)

	// from address is either default or specified on communication
	senderPubKey := s.pubKeys[0]
	senderPrivKey := s.privKeys[0]

	encryptedPayload := api.EncryptedPayload {
		Sender:         senderPubKey,
		CipherText:     sealedMessage,
		Nonce:          nonce[:],
		RecipientBoxes: make([][]byte, len(recipients)),
		RecipientNonce: recipientNonce,
	}

	for _, recipient := range recipients {
		if url, ok := s.partyInfo.Recipients[recipient]; ok {

			recipientKey, err := LoadKey(recipient)
			if err != nil {
				log.WithField("recipientKey", recipientKey).Errorf(
					"Unable to load recipient, %v", err)
			}

			sealedBox := sealPayload(recipientNonce, masterKey, recipientKey, senderPrivKey)
			encryptedPayload.RecipientBoxes = [][]byte{ sealedBox }
			Push(encryptedPayload, url)
		} else {
			log.WithField("recipientKey", recipient).Error("Unable to resolve host")
		}
	}

	sha3Hash := sha3.New512()
	sha3Hash.Write(sealedMessage)
	digestHash := sha3Hash.Sum(nil)

	encodedPayload := sealPayload(recipientNonce, masterKey, senderPubKey, senderPrivKey)
	err := s.Db.Write(&digestHash, &encodedPayload)
	return digestHash, err
}

func sealPayload(
	recipientNonce nacl.Nonce,
	masterKey nacl.Key,
	recipientKey nacl.Key,
	privateKey nacl.Key) []byte {

	return box.Seal(
		[]byte{},
		(*masterKey)[:],
		recipientNonce,
		recipientKey,
		privateKey)
}

func Push(epl api.EncryptedPayload, url string) (string, error) {

	encodedPl := api.EncodePayload(epl)

	resp, err := http.Post(
		url + "/push", "application/octet-stream", bytes.NewReader(encodedPl))
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

func (s *Enclave) Retrieve(key nacl.Key, digestHash *[]byte) ([]byte, error) {
	digest, err := s.Db.Read(digestHash)
	if err != nil {
		return nil, err
	} else {
		return secretbox.EasyOpen(*digest, key)
	}
}

func (s *Enclave) Delete(digestHash *[]byte) error {
	return s.Db.Delete(digestHash)
}

func LoadKey(hexkey string) (nacl.Key, error) {
	return nacl.Load(hexkey)
}

func NewKey() nacl.Key {
	return nacl.NewKey()
}
