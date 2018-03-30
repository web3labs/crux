package enclave

import (
	"github.com/kevinburke/nacl"
	"github.com/kevinburke/nacl/box"
	"github.com/kevinburke/nacl/secretbox"
	"gitlab.com/blk-io/crux/storage"
	"golang.org/x/crypto/sha3"
	"gitlab.com/blk-io/crux/api"
	log "github.com/sirupsen/logrus"
	"errors"
	"bytes"
	"fmt"
)

type Enclave struct {
	Db         storage.DataStore
	PubKeys    []nacl.Key
	PrivKeys   []nacl.Key
	selfPubKey nacl.Key
	PartyInfo  api.PartyInfo
	keyCache   map[nacl.Key]map[nacl.Key]nacl.Key  // maps sender -> recipient -> shared key
}

func (s *Enclave) Init() {

	s.keyCache = make(map[nacl.Key]map[nacl.Key]nacl.Key)

	s.selfPubKey = nacl.NewKey()

	for _, pubKey := range s.PubKeys {
		s.keyCache[pubKey] = make(map[nacl.Key]nacl.Key)

		// We have a once off generated key which we use for storing payloads which are addressed
		// only to ourselves. We have to do this, as we cannot use box.Seal with a public and
		// private keypair
		//
		// We precompute these keys on startup
		s.resolveSharedKey(s.PrivKeys[0], pubKey, s.selfPubKey)
	}

	/*
	We use shared keys for encrypting data. The keys between a specific sender and recipient are
	computed once for each unique

	Encrypt scenarios:
	The sender value must always be a public key that we have the corresponding private key for
	privateFor: [] => 	encrypt with sharedKey [self-private, selfPub-public]
						store in cache as (self-public, selfPub-public)
	privateFor: [recipient1, ...] => encrypt with sharedKey1 [self-private, recipient1-public], ...
						store in cache as (self-public, recipient1-public)
	Decrypt scenarios:
	epl, [] => The payload was pushed to us (we are recipient1), decrypt with sharedKey [recipient1-private, sender-public]
						lookup in cache as (recipient1-public, sender-public)
	epl, [recipient1, ...,] => The payload originated with us (we are self), decrypt with sharedKey [self-private, recipient1-public]
						lookup in cache as (self-public, recipient1-public)

	Note that sharedKey(privA, pubB) produces the same key as sharedKey(pubA, privB), which is why
	when sending to ones self we encrypt with sharedKey [self-private, selfPub-public], then
	retrieve with sharedKey [self-private, selfPub-public]
	 */
}

func (s *Enclave) Store(
	message *[]byte, sender string, recipients []string) ([]byte, error) {

		var senderPubKey, senderPrivKey nacl.Key

		if sender == "" {
			// from address is either default or specified on communication
			senderPubKey = s.PubKeys[0]
			senderPrivKey = s.PrivKeys[0]
		}

		senderPubKey, err := nacl.Load(sender)
		if err != nil {
			log.WithField("senderPubKey", sender).Errorf(
				"Unable to load sender public key, %v", err)
			return nil, err
		}

		senderPrivKey, err = s.resolvePrivateKey(senderPubKey)
		if err != nil {
			log.WithField("senderPubKey", sender).Errorf(
				"Unable to locate private key for sender public key, %v", err)
			return nil, err
		}

		return s.store(message, senderPubKey, senderPrivKey, recipients)
	}

func (s *Enclave) store(
	message *[]byte,
	senderPubKey, senderPrivKey nacl.Key,
	recipients []string) ([]byte, error) {

	nonce := nacl.NewNonce()
	masterKey := nacl.NewKey()
	recipientNonce := nacl.NewNonce()

	sealedMessage := secretbox.Seal([]byte{}, *message, nonce, masterKey)

	encryptedPayload := api.EncryptedPayload {
		Sender:         senderPubKey,
		CipherText:     sealedMessage,
		Nonce:          nonce,
		RecipientBoxes: make([][]byte, len(recipients)),
		RecipientNonce: recipientNonce,
	}

	recipientsSlice := make([][]byte, len(recipients))

	for i, recipient := range recipients {

		recipientKey, err := nacl.Load(recipient)
		if err != nil {
			log.WithField("recipientKey", recipientKey).Errorf(
				"Unable to load recipient, %v", err)
		}

		// TODO: We may what to loosen this check
		if bytes.Equal((*recipientKey)[:], (*senderPubKey)[:]) {
			log.WithField("recipientKey", recipientKey).Errorf(
				"Sender cannot be recipient, %v", err)
		}

		sharedKey := s.resolveSharedKey(senderPrivKey, senderPubKey, recipientKey)
		sealedBox := sealPayload(recipientNonce, masterKey, sharedKey)

		encryptedPayload.RecipientBoxes[i] = sealedBox
		recipientsSlice[i] = []byte(recipient)
	}

	// store locally
	sharedKey := s.resolveSharedKey(senderPrivKey, senderPubKey, s.selfPubKey)

	sealedBox := sealPayload(recipientNonce, masterKey, sharedKey)
	encryptedPayload.RecipientBoxes = [][]byte{ sealedBox }

	encodedEpl := api.EncodePayloadWithRecipients(encryptedPayload, recipientsSlice)
	digest, err := s.storePayload(encryptedPayload, encodedEpl)

	for i, recipient := range recipients {
		recipientEpl := api.EncryptedPayload{
			Sender:         senderPubKey,
			CipherText:     sealedMessage,
			Nonce:          nonce,
			RecipientBoxes: [][]byte{encryptedPayload.RecipientBoxes[i]},
			RecipientNonce: recipientNonce,
		}

		s.publishPayload(recipientEpl, recipient)
	}

	return digest, err
}

func (s *Enclave) publishPayload(epl api.EncryptedPayload, recipient string) {

	if url, ok := s.PartyInfo.Recipients[recipient]; ok {

		encoded := api.EncodePayloadWithRecipients(epl, [][]byte{})
		api.Push(encoded, url)
	} else {
		log.WithField("recipientKey", recipient).Error("Unable to resolve host")
	}
}

func (s *Enclave) resolveSharedKey(senderPrivKey, senderPubKey, recipientPubKey nacl.Key) nacl.Key {

	keyCache := s.keyCache[senderPubKey]

	sharedKey, ok := keyCache[recipientPubKey]
	if !ok {
		sharedKey = box.Precompute(recipientPubKey, senderPrivKey)
		keyCache[recipientPubKey] = sharedKey
	}

	return sharedKey
}

func (s *Enclave) resolvePrivateKey(publicKey nacl.Key) (nacl.Key, error) {
	for i, key := range s.PubKeys {
		if bytes.Equal((*publicKey)[:], (*key)[:]) {
			return s.PrivKeys[i], nil
		}
	}
	return nil, errors.New("unable to find private key for public key")
}

func (s *Enclave) StorePayload(encoded []byte) ([]byte, error) {
	epl, _ := api.DecodePayloadWithRecipients(encoded)
	return s.storePayload(epl, encoded)
}

func (s *Enclave) storePayload(epl api.EncryptedPayload, encoded []byte) ([]byte, error) {

	sha3Hash := sha3.New512()
	sha3Hash.Write(epl.CipherText)
	digestHash := sha3Hash.Sum(nil)

	err := s.Db.Write(&digestHash, &encoded)
	return digestHash, err
}

func sealPayload(
	recipientNonce nacl.Nonce,
	masterKey nacl.Key,
	sharedKey nacl.Key) []byte {

	return box.SealAfterPrecomputation(
		[]byte{},
		(*masterKey)[:],
		recipientNonce,
		sharedKey)
}

func (s *Enclave) Retrieve(digestHash *[]byte, to *[]byte) ([]byte, error) {

	encoded, err := s.Db.Read(digestHash)
	if err != nil {
		return nil, err
	}

	epl, recipients := api.DecodePayloadWithRecipients(*encoded)

	masterKey := new([nacl.KeySize]byte)

	var senderPubKey, senderPrivKey, recipientPubKey, sharedKey nacl.Key

	if len(recipients) == 0 {
		// This is a payload originally sent to us by another node
		recipientPubKey = epl.Sender
		senderPubKey, err = toKey(*to)
		if err != nil {
			return nil, err
		}
	} else {
		// This is a payload that originated from us
		senderPubKey = epl.Sender
		recipientPubKey, err = toKey(recipients[0])
		if err != nil {
			return nil, err
		}
	}

	senderPrivKey, err = s.resolvePrivateKey(senderPubKey)
	if err != nil {
		return nil, err
	}

	// we might not have the key in our cache if constellation was restarted, hence we may
	// need to recreate
	sharedKey = s.resolveSharedKey(senderPrivKey, senderPubKey, recipientPubKey)

	_, ok := secretbox.Open(masterKey[:], epl.RecipientBoxes[0], epl.RecipientNonce, sharedKey)
	if !ok {
		return nil, errors.New("unable to open master key secret box")
	}

	payload := make([]byte, len(epl.CipherText))
	_, ok = secretbox.Open(nil, epl.CipherText, epl.Nonce, masterKey)
	if !ok {
		return payload, errors.New("unable to open payload secret box")
	}

	return payload, nil
}

func (s *Enclave) Delete(digestHash *[]byte) error {
	return s.Db.Delete(digestHash)
}

func (s *Enclave) UpdatePartyInfo(encoded []byte) {
	pi := api.DecodePartyInfo(encoded)

	for publicKey, url := range pi.Recipients {
		// we should ignore messages about ourselves
		// in order to stop people masquerading as you, there
		// should be a digital signature associated with each
		// url -> node broadcast
		if url != s.PartyInfo.Url {
			s.PartyInfo.Recipients[publicKey] = url
		}
	}

	for url := range pi.Parties {
		// we don't want to broadcast party info to ourselves
		if url != s.PartyInfo.Url {
			s.PartyInfo.Parties[url] = true
		}
	}
}

func toKey(src []byte) (nacl.Key, error) {
	if len(src) != nacl.KeySize {
		return nil, fmt.Errorf("nacl: incorrect key length: %d", len(src))
	}
	key := new([nacl.KeySize]byte)
	copy(key[:], src)
	return key, nil
}
