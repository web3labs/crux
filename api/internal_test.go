package api

import (
	"testing"
	"github.com/kevinburke/nacl"
	"reflect"
)

func TestEncodePayload(t *testing.T) {

	epl := EncryptedPayload{
		Sender: nacl.NewKey(),
		CipherText: []byte("C1ph3r T3xt"),
		Nonce: nacl.NewNonce(),
		RecipientBoxes: [][]byte{ []byte("B0x1"), []byte("B0x2") },
		RecipientNonce: nacl.NewNonce(),
	}

	encoded := EncodePayload(epl)
	decoded, err := DecodePayload(encoded)

	if err != nil ||
		!reflect.DeepEqual(epl, decoded) {
		t.Errorf("Decoded payload: %v does not match input %v", decoded, epl)
	}
}
