// Package enclave provides enclaves for the secure storage and propagation of transactions.
package enclave

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/blk-io/crux/api"
	"github.com/blk-io/crux/storage"
	"github.com/blk-io/crux/utils"
	"github.com/kevinburke/nacl"
	"github.com/kevinburke/nacl/box"
	"github.com/kevinburke/nacl/secretbox"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"path/filepath"
	"strings"
)

// SecureEnclave is the secure transaction enclave.
type SecureEnclave struct {
	Db         storage.DataStore                  // The underlying key-value datastore for encrypted transactions
	PubKeys    []nacl.Key                         // Public keys associated with this enclave
	PrivKeys   []nacl.Key                         // Private keys associated with this enclave
	selfPubKey nacl.Key                           // An ephemeral key used for transactions only intended for this enclave
	PartyInfo  api.PartyInfo                      // Details of all other nodes (or parties) on the network
	keyCache   map[nacl.Key]map[nacl.Key]nacl.Key // Maps sender -> recipient -> shared key
	client     utils.HttpClient                   // The underlying HTTP client used to propagate requests
	grpc       bool
}

// Init creates a new instance of the SecureEnclave.
func Init(
	db storage.DataStore,
	pubKeyFiles, privKeyFiles []string,
	pi api.PartyInfo,
	client utils.HttpClient, grpc bool) *SecureEnclave {

	// Key format:
	// BULeR8JyUWhiuuCMU/HLA0Q5pzkYT+cHII3ZKBey3Bo=
	pubKeys, err := loadPubKeys(pubKeyFiles)
	if err != nil {
		log.Fatalf("Unable to load public key files: %s, error: %v", pubKeyFiles, err)
	}

	// Key format:
	// {"data":{"bytes":"Wl+xSyXVuuqzpvznOS7dOobhcn4C5auxkFRi7yLtgtA="},"type":"unlocked"}
	privKeys, err := loadPrivKeys(privKeyFiles)
	if err != nil {
		log.Fatalf("Unable to load private key files: %s, error: %v", privKeyFiles, err)
	}

	enc := SecureEnclave{
		Db:        db,
		PubKeys:   pubKeys,
		PrivKeys:  privKeys,
		PartyInfo: pi,
		client:    client,
		grpc:      grpc,
	}

	// We use shared keys for encrypting data. The keys between a specific sender and recipient are
	// computed once for each unique pair.
	//
	// Encrypt scenarios:
	// The sender value must always be a public key that we have the corresponding private key for
	// privateFor: [] => 	encrypt with sharedKey [self-private, selfPub-public]
	// 		store in cache as (self-public, selfPub-public)
	// privateFor: [recipient1, ...] => encrypt with sharedKey1 [self-private, recipient1-public], ...
	//     store in cache as (self-public, recipient1-public)
	// Decrypt scenarios:
	// epl, [] => The payload was pushed to us (we are recipient1), decrypt with sharedKey
	//     [recipient1-private, sender-public]
	// 	   lookup in cache as (recipient1-public, sender-public)
	// epl, [recipient1, ...,] => The payload originated with us (we are self), decrypt with
	//     sharedKey [self-private, recipient1-public]
	//     lookup in cache as (self-public, recipient1-public)
	//
	// Note that sharedKey(privA, pubB) produces the same key as sharedKey(pubA, privB), which is
	// why when sending to ones self we encrypt with sharedKey [self-private, selfPub-public], then
	// retrieve with sharedKey [self-private, selfPub-public]
	enc.keyCache = make(map[nacl.Key]map[nacl.Key]nacl.Key)

	enc.selfPubKey = nacl.NewKey()

	for _, pubKey := range enc.PubKeys {
		enc.keyCache[pubKey] = make(map[nacl.Key]nacl.Key)

		// We have a once off generated key which we use for storing payloads which are addressed
		// only to ourselves. We have to do this, as we cannot use box.Seal with a public and
		// private key-pair.
		//
		// We pre-compute these keys on startup.
		enc.resolveSharedKey(enc.PrivKeys[0], pubKey, enc.selfPubKey)
	}

	return &enc
}

// Store a payload submitted via an Ethereum node.
// This function encrypts the payload, and distributes the encrypted payload to the other
// specified recipients in the network.
// The hash of the encrypted payload is returned to the sender.
func (s *SecureEnclave) Store(
	message *[]byte, sender []byte, recipients [][]byte) ([]byte, error) {

	var err error
	var senderPubKey, senderPrivKey nacl.Key

	if len(sender) == 0 {
		// from address is either default or specified on communication
		senderPubKey = s.PubKeys[0]
		senderPrivKey = s.PrivKeys[0]
	} else {
		senderPubKey, err = utils.ToKey(sender)
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
	}

	return s.store(message, senderPubKey, senderPrivKey, recipients)
}

func (s *SecureEnclave) store(
	message *[]byte,
	senderPubKey, senderPrivKey nacl.Key,
	recipients [][]byte) ([]byte, error) {

	epl, masterKey := createEncryptedPayload(message, senderPubKey, recipients)

	for i, recipient := range recipients {

		recipientKey, err := utils.ToKey(recipient)
		if err != nil {
			log.WithField("recipientKey", recipientKey).Errorf(
				"Unable to load recipient, %v", err)
			continue
		}

		// TODO: We may choose to loosen this check
		if bytes.Equal((*recipientKey)[:], (*senderPubKey)[:]) {
			log.WithField("recipientKey", recipientKey).Errorf(
				"Sender cannot be recipient, %v", err)
			continue
		}

		sharedKey := s.resolveSharedKey(senderPrivKey, senderPubKey, recipientKey)
		sealedBox := sealPayload(epl.RecipientNonce, masterKey, sharedKey)

		epl.RecipientBoxes[i] = sealedBox
	}

	var toSelf bool
	if len(recipients) == 0 {
		toSelf = true
		recipients = [][]byte{(*s.selfPubKey)[:]}
	} else {
		toSelf = false
	}

	// store locally
	recipientKey, err := utils.ToKey(recipients[0])
	if err != nil {
		log.WithField("recipientKey", recipientKey).Errorf(
			"Unable to load recipient, %v", err)
	}

	sharedKey := s.resolveSharedKey(senderPrivKey, senderPubKey, recipientKey)

	sealedBox := sealPayload(epl.RecipientNonce, masterKey, sharedKey)
	epl.RecipientBoxes = [][]byte{sealedBox}

	encodedEpl := api.EncodePayloadWithRecipients(epl, recipients)
	digest, err := s.storePayload(epl, encodedEpl)

	if !toSelf {
		for i, recipient := range recipients {
			recipientEpl := api.EncryptedPayload{
				Sender:         senderPubKey,
				CipherText:     epl.CipherText,
				Nonce:          epl.Nonce,
				RecipientBoxes: [][]byte{epl.RecipientBoxes[i]},
				RecipientNonce: epl.RecipientNonce,
			}

			log.WithFields(log.Fields{
				"recipient": hex.EncodeToString(recipient), "digest": hex.EncodeToString(digest),
			}).Debug("Publishing payload")

			s.publishPayload(recipientEpl, recipient)
		}
	}

	return digest, err
}

func createEncryptedPayload(
	message *[]byte, senderPubKey nacl.Key, recipients [][]byte) (api.EncryptedPayload, nacl.Key) {

	nonce := nacl.NewNonce()
	masterKey := nacl.NewKey()
	recipientNonce := nacl.NewNonce()

	sealedMessage := secretbox.Seal([]byte{}, *message, nonce, masterKey)

	return api.EncryptedPayload{
		Sender:         senderPubKey,
		CipherText:     sealedMessage,
		Nonce:          nonce,
		RecipientBoxes: make([][]byte, len(recipients)),
		RecipientNonce: recipientNonce,
	}, masterKey
}

func (s *SecureEnclave) publishPayload(epl api.EncryptedPayload, recipient []byte) {

	key, err := utils.ToKey(recipient)
	if err != nil {
		log.WithField("recipient", recipient).Errorf(
			"Unable to decode key for recipient, error: %v", err)
	}

	if url, ok := s.PartyInfo.GetRecipient(key); ok {
		encoded := api.EncodePayloadWithRecipients(epl, [][]byte{})
		if s.grpc {
			api.PushGrpc(encoded, url, epl)
		} else {
			api.Push(encoded, url, s.client)
		}
	} else {
		log.WithField("recipientKey", hex.EncodeToString(recipient)).Error("Unable to resolve host")
	}
}

func (s *SecureEnclave) resolveSharedKey(
	senderPrivKey, senderPubKey, recipientPubKey nacl.Key) nacl.Key {

	keyCache, ok := s.keyCache[senderPubKey]
	if !ok {
		keyCache = make(map[nacl.Key]nacl.Key)
		s.keyCache[senderPubKey] = keyCache
	}

	sharedKey, ok := keyCache[recipientPubKey]
	if !ok {
		sharedKey = box.Precompute(recipientPubKey, senderPrivKey)
		keyCache[recipientPubKey] = sharedKey
	}

	return sharedKey
}

func (s *SecureEnclave) resolvePrivateKey(publicKey nacl.Key) (nacl.Key, error) {
	for i, key := range s.PubKeys {
		if bytes.Equal((*publicKey)[:], (*key)[:]) {
			return s.PrivKeys[i], nil
		}
	}
	return nil, fmt.Errorf("unable to find private key for public key: %s",
		hex.EncodeToString((*publicKey)[:]))
}

// Store a binary encoded payload within this SecureEnclave.
// This will be a payload that has been propagated to this node as it is a party on the
// transaction. I.e. it is not the original recipient of the transaction, but one of the recipients
// it is intended for.
func (s *SecureEnclave) StorePayload(encoded []byte) ([]byte, error) {
	epl, _ := api.DecodePayloadWithRecipients(encoded)
	return s.storePayload(epl, encoded)
}

func (s *SecureEnclave) StorePayloadGrpc(epl api.EncryptedPayload, encoded []byte) ([]byte, error) {
	return s.storePayload(epl, encoded)
}

func (s *SecureEnclave) storePayload(epl api.EncryptedPayload, encoded []byte) ([]byte, error) {
	digestHash := utils.Sha3Hash(epl.CipherText)
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

// RetrieveDefault is used to retrieve the provided payload. It attempts to use a default key
// value of the first public key associated with this SecureEnclave instance.
// If the payload cannot be found, or decrypted successfully an error is returned.
func (s *SecureEnclave) RetrieveDefault(digestHash *[]byte) ([]byte, error) {
	// to address is either default or specified on communication
	key := (*s.PubKeys[0])[:]
	return s.Retrieve(digestHash, &key)
}

// Retrieve is used to retrieve the provided payload.
// If the payload cannot be found, or decrypted successfully an error is returned.
func (s *SecureEnclave) Retrieve(digestHash *[]byte, to *[]byte) ([]byte, error) {

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
		senderPubKey, err = utils.ToKey(*to)
		if err != nil {
			return nil, err
		}
	} else {
		// This is a payload that originated from us
		senderPubKey = epl.Sender
		recipientPubKey, err = utils.ToKey(recipients[0])
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

	_, ok := secretbox.Open(masterKey[:0], epl.RecipientBoxes[0], epl.RecipientNonce, sharedKey)
	if !ok {
		return nil, errors.New("unable to open master key secret box")
	}

	var payload []byte
	payload, ok = secretbox.Open(payload[:0], epl.CipherText, epl.Nonce, masterKey)
	if !ok {
		return payload, errors.New("unable to open payload secret box")
	}

	return payload, nil
}

// RetrieveFor retrieves a payload with the given digestHash for a specific recipient who was one
// of the original recipients specified on the payload.
func (s *SecureEnclave) RetrieveFor(digestHash *[]byte, reqRecipient *[]byte) (*[]byte, error) {
	encoded, err := s.Db.Read(digestHash)
	if err != nil {
		return nil, err
	}

	epl, recipients := api.DecodePayloadWithRecipients(*encoded)

	for i, recipient := range recipients {
		if bytes.Equal(*reqRecipient, recipient) {
			recipientEpl := api.EncryptedPayload{
				Sender:         epl.Sender,
				CipherText:     epl.CipherText,
				Nonce:          epl.Nonce,
				RecipientBoxes: [][]byte{epl.RecipientBoxes[i]},
				RecipientNonce: epl.RecipientNonce,
			}
			encoded := api.EncodePayload(recipientEpl)
			return &encoded, nil
		}
	}
	return nil, fmt.Errorf("invalid recipient %x requested for payload", reqRecipient)
}

// RetrieveAllFor retrieves all payloads that the specified recipient was an original recipient
// for.
// Each payload found is published to the specified recipient.
func (s *SecureEnclave) RetrieveAllFor(reqRecipient *[]byte) error {
	return s.Db.ReadAll(func(key, value *[]byte) {
		epl, recipients := api.DecodePayloadWithRecipients(*value)

		for i, recipient := range recipients {
			if bytes.Equal(*reqRecipient, recipient) {
				recipientEpl := api.EncryptedPayload{
					Sender:         epl.Sender,
					CipherText:     epl.CipherText,
					Nonce:          epl.Nonce,
					RecipientBoxes: [][]byte{epl.RecipientBoxes[i]},
					RecipientNonce: epl.RecipientNonce,
				}
				func() {
					go s.publishPayload(recipientEpl, *reqRecipient)
				}()
			}
		}
	})
}

// Delete deletes the payload associated with the given digestHash from the SecureEnclave's store.
func (s *SecureEnclave) Delete(digestHash *[]byte) error {
	return s.Db.Delete(digestHash)
}

// UpdatePartyInfo applies the provided binary encoded party details to the SecureEnclave's
// own party details store.
func (s *SecureEnclave) UpdatePartyInfo(encoded []byte) {
	s.PartyInfo.UpdatePartyInfo(encoded)
}

func (s *SecureEnclave) UpdatePartyInfoGrpc(url string, recipients map[[nacl.KeySize]byte]string, parties map[string]bool) {
	s.PartyInfo.UpdatePartyInfoGrpc(url, recipients, parties)
}

// GetEncodedPartyInfo provides this SecureEnclaves PartyInfo details in a binary encoded format.
func (s *SecureEnclave) GetEncodedPartyInfo() []byte {
	return api.EncodePartyInfo(s.PartyInfo)
}

func (s *SecureEnclave) GetEncodedPartyInfoGrpc() []byte {
	encoded, err := json.Marshal(api.PartyInfoResponse{Payload: api.EncodePartyInfo(s.PartyInfo)})
	if err != nil {
		log.Errorf("Marshalling failed %v", err)
	}
	return encoded
}

func (s *SecureEnclave) GetPartyInfo() (string, map[[nacl.KeySize]byte]string, map[string]bool) {
	return s.PartyInfo.GetAllValues()
}

func loadPubKeys(pubKeyFiles []string) ([]nacl.Key, error) {
	return loadKeys(
		pubKeyFiles,
		func(s string) (string, error) {
			src, err := ioutil.ReadFile(s)
			if err != nil {
				return "", err
			}
			return string(src), nil
		})
}

func loadPrivKeys(privKeyFiles []string) ([]nacl.Key, error) {
	return loadKeys(
		privKeyFiles,
		func(s string) (string, error) {
			var privateKey api.PrivateKey
			src, err := ioutil.ReadFile(s)
			if err != nil {
				return "", err
			}
			err = json.Unmarshal(src, &privateKey)
			if err != nil {
				return "", err
			}

			return privateKey.Data.Bytes, nil
		})
}

func loadKeys(
	keyFiles []string, f func(string) (string, error)) ([]nacl.Key, error) {
	keys := make([]nacl.Key, len(keyFiles))

	for i, keyFile := range keyFiles {
		data, err := f(keyFile)
		if err != nil {
			return nil, err
		}
		var key nacl.Key
		key, err = utils.LoadBase64Key(
			strings.TrimSuffix(data, "\n"))
		if err != nil {
			return nil, err
		}
		keys[i] = key
	}

	return keys, nil
}

// DoKeyGeneration is used to generate new public and private key-pairs, writing them to the
// provided file locations.
// Public keys have the "pub" suffix, whereas private keys have the "key" suffix.
func DoKeyGeneration(keyFile string) error {
	pubKey, privKey, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("error creating keys: %v", err)
	}
	err = utils.CreateDirForFile(keyFile)
	if err != nil {
		return fmt.Errorf("invalid destination specified: %s, error: %v",
			filepath.Dir(keyFile), err)
	}

	b64PubKey := base64.StdEncoding.EncodeToString((*pubKey)[:])
	b64PrivKey := base64.StdEncoding.EncodeToString((*privKey)[:])

	err = ioutil.WriteFile(keyFile+".pub", []byte(b64PubKey), 0600)
	if err != nil {
		return fmt.Errorf("unable to write public key: %s, error: %v", keyFile, err)
	}

	jsonKey := api.PrivateKey{
		Type: "unlocked",
		Data: api.PrivateKeyBytes{
			Bytes: b64PrivKey,
		},
	}

	var encoded []byte
	encoded, err = json.Marshal(jsonKey)
	if err != nil {
		return fmt.Errorf("unable to encode private key: %v, error: %v", jsonKey, err)
	}

	err = ioutil.WriteFile(keyFile+".key", encoded, 0600)
	if err != nil {
		return fmt.Errorf("unable to write private key: %s, error: %v", keyFile, err)
	}
	return nil
}
