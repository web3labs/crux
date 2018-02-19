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

func (s *Enclave) Store(key nacl.Key, message *[]byte) ([]byte, error) {
	digest := secretbox.EasySeal(*message, key)

	sha3Hash := sha3.New512()
	sha3Hash.Write(digest)
	digestHash := sha3Hash.Sum(nil)

	err := s.Db.Write(&digestHash, &digest)
	return digestHash, err
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
