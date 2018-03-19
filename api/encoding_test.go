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
	decoded := DecodePayload(encoded)

	if !reflect.DeepEqual(epl, decoded) {
		t.Errorf("Decoded payload: %v does not match input %v", decoded, epl)
	}
}


func TestEncodePartyInfo(t *testing.T) {

	pi := PartyInfo{
		Url: "https://127.0.0.4:9004/",
		Recipients: map[string]string{
			"ROAZBWtSacxXQrOe3FGAqJDyJjFePR5ce4TSIzmJ0Bc=": "https://127.0.0.7:9007/",
			"BULeR8JyUWhiuuCMU/HLA0Q5pzkYT+cHII3ZKBey3Bo=": "https://127.0.0.1:9001/",
			"QfeDAys9MPDs2XHExtc84jKGHxZg/aj52DTh0vtA3Xc=": "https://127.0.0.2:9002/",
			"1iTZde/ndBHvzhcl7V68x44Vx7pl8nwx9LqnM/AfJUg=": "https://127.0.0.3:9003/",
			"UfNSeSGySeKg11DVNEnqrUtxYRVor4+CvluI8tVv62Y=": "https://127.0.0.6:9006/",
			"oNspPPgszVUFw0qmGFfWwh1uxVUXgvBxleXORHj07g8=": "https://127.0.0.4:9004/",
			"R56gy4dn24YOjwyesTczYa8m5xhP6hF2uTMCju/1xkY=": "https://127.0.0.5:9005/",
		},
		Parties: map[string]bool{
			"https://127.0.0.5:9005/": true,
			"https://127.0.0.3:9003/": true,
			"https://127.0.0.1:9001/": true,
			"https://127.0.0.7:9007/": true,
			"https://127.0.0.6:9006/": true,
			"https://127.0.0.4:9004/": true,
			"https://127.0.0.2:9002/": true,
		},
	}

	encoded := EncodePartyInfo(pi)
	decoded := DecodePartyInfo(encoded)

	if !reflect.DeepEqual(pi, decoded) {
		t.Errorf("Decoded partyInfo: %v does not match input %v", decoded, pi)
	}
}