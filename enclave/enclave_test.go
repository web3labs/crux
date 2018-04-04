package enclave

import (
	"testing"
	"io/ioutil"
	"gitlab.com/blk-io/crux/storage"
	"gitlab.com/blk-io/crux/api"
	"bytes"
	"os"
)

var message = []byte("Test message")
const tempDir = "enclaveTest"

func TestMain(m *testing.M) {
	retCode := m.Run()
	cleanUp()
	os.Exit(retCode)
}

func cleanUp() {

}

func initEnclave(t *testing.T, name string) Enclave {
	dbPath, err := ioutil.TempDir("", name)
	if err != nil {
		t.Fatal(err)
	}

	db, err := storage.Init(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	pi := api.LoadPartyInfo(
		"http://localhost:8000",
		[]string{"http://localhost:8000"})

	return Init(
		db,
		[]string{"testdata/key.pub"},
		[]string{"testdata/key"},
		pi)
}

func TestStoreAndRetrieve(t *testing.T) {
	enc := initEnclave(t, "TestStoreAndRetrieve")

	pubKeys, err := loadPubKeys([]string{"testdata/rcpt1.pub"})
	rcpt1 := pubKeys[0]

	var digest []byte
	digest, err = enc.Store(&message, "", []string{string((*rcpt1)[:])})
	if err != nil {
		t.Fatal(err)
	}

	var returned []byte
	returned, err = enc.Retrieve(&digest, nil)

	if !bytes.Equal(message, returned) {
		t.Errorf(
			"Retrieved message is not the same as original:\n" +
				"Original: %v\nRetrieved: %v",
			message, returned)
	}
}

func TestStoreAndRetrieveSelf(t *testing.T) {
	enc := initEnclave(t, "TestStoreAndRetrieveSelf")

	digest, err := enc.Store(&message, "", []string{})
	if err != nil {
		t.Fatal(err)
	}

	var returned []byte
	returned, err = enc.Retrieve(&digest, nil)

	if !bytes.Equal(message, returned) {
		t.Errorf(
			"Retrieved message is not the same as original:\n" +
				"Original: %v\nRetrieved: %v",
			message, returned)
	}
}

func TestStoreNotAuthorised(t *testing.T) {

}

func TestRetrieveNotAuthorised(t *testing.T) {

}
