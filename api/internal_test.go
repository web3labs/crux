package api

import (
	"testing"
	"net/http"
	"github.com/kevinburke/nacl"
)

func TestRegisterPublicKeys(t *testing.T) {
	key := []nacl.Key{nacl.NewKey()}

	pi := CreatePartyInfo(
		"http://localhost:9000",
		[]string{"http://localhost:9001"},
		key,
		http.DefaultClient)

	expKey := []nacl.Key{nacl.NewKey()}
	expUrl := "http://localhost:9000"

	pi.RegisterPublicKeys(expKey)

	url, ok := pi.GetRecipient(expKey[0])
	if !ok || url != expUrl{
		t.Errorf("Url is %s whereas %s is expected", url, expUrl)
	}

}
