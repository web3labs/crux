package enclave

import (
	"github.com/kevinburke/nacl"
	"github.com/kevinburke/nacl/secretbox"
	"github.com/blk-io/crux/storage"
	"golang.org/x/crypto/sha3"
)

type Enclave struct {
	Db storage.DataStore
}

func (s *Enclave) Store(key nacl.Key, message *[]byte) []byte {
	digest := secretbox.EasySeal(*message, key)

	sha3Hash := sha3.New512()
	sha3Hash.Write(digest)
	digestHash := sha3Hash.Sum(nil)

	s.Db.Write(&digestHash, &digest)
	return digestHash
}

func (s *Enclave) Retrieve(key nacl.Key, digestHash *[]byte) []byte {
	digest := s.Db.Read(digestHash)

	payload, err := secretbox.EasyOpen(*digest, key)
	if err != nil {
		panic(err)
	}

	return payload
}

func (s *Enclave) Delete(digestHash *[]byte) {
	s.Db.Delete(digestHash)
}

func LoadKey(hexkey string) nacl.Key {
	key, err := nacl.Load(hexkey)
	if err != nil {
		panic(err)
	}
	return key
}

func NewKey() nacl.Key {
	return nacl.NewKey()
}