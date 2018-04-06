package utils

import "golang.org/x/crypto/sha3"

func Sha3Hash(payload []byte) []byte {
	sha3Hash := sha3.New512()
	sha3Hash.Write(payload)
	return sha3Hash.Sum(nil)
}
