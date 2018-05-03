package enclave

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sync"
	"testing"
	"time"
	"github.com/blk-io/crux/storage"
	"github.com/blk-io/crux/api"
	"github.com/blk-io/crux/utils"
	"github.com/kevinburke/nacl"
)

var message = []byte("Test message")

type MockClient struct {
	serviceMu sync.Mutex
	requests [][]byte
}

func (c *MockClient) Do(req *http.Request) (*http.Response, error) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	c.serviceMu.Lock()
	c.requests = append(c.requests, body)
	c.serviceMu.Unlock()

	respBody := ioutil.NopCloser(bytes.NewReader([]byte("")))
	return &http.Response{Body: respBody}, nil
}

func (c *MockClient) reqCount() int {
	c.serviceMu.Lock()
	defer c.serviceMu.Unlock()
	return len(c.requests)
}

func initEnclave(
	t *testing.T,
	dbPath string,
	pi api.PartyInfo,
	client utils.HttpClient) *SecureEnclave {

	db, err := storage.InitLevelDb(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	return Init(
		db,
		[]string{"testdata/key.pub"},
		[]string{"testdata/key"},
		pi,
		client)
}

func initDefaultEnclave(t *testing.T,
	dbPath string) *SecureEnclave {

	var client utils.HttpClient
	client = &MockClient{}
	pi := api.InitPartyInfo(
		"http://localhost:8000",
		[]string{"http://localhost:8001"}, client)

	return initEnclave(t, dbPath, pi, client)
}

func TestStoreAndRetrieve(t *testing.T) {
	dbPath, err := ioutil.TempDir("", "TestStoreAndRetrieve")

	if err != nil {
		t.Fatal(err)
	} else {
		defer os.RemoveAll(dbPath)
	}

	mockClient := &MockClient{requests: [][]byte{}}
	var client utils.HttpClient
	client = mockClient

	pubKeys, err := loadPubKeys([]string{"testdata/rcpt1.pub"})
	if err != nil {
		t.Fatal(err)
	}
	rcpt1 := pubKeys[0]

	pi := api.CreatePartyInfo(
		"http://localhost:8000",
		[]string{"http://localhost:8001"},
		[]nacl.Key{rcpt1},
		client)

	enc := initEnclave(t, dbPath, pi, client)

	var digest []byte
	digest, err = enc.Store(&message, []byte{}, [][]byte{(*rcpt1)[:]})
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

	// We verify payload propagation too
	if mockClient.reqCount() != 1 {
		t.Errorf("Only one request should have been captured, actual: %d\n",
			len(mockClient.requests))
	}

	propagatedPl := mockClient.requests[0]
	epl, recipients := api.DecodePayloadWithRecipients(propagatedPl)

	if len(recipients) != 0 {
		t.Errorf("Recipients should be empty in data sent to other nodes, actual size: %d\n",
			len(recipients))
	}

	if len(epl.RecipientBoxes) != 1 {
		t.Errorf("There should only be one recipient box present, actual %d\n",
			len(epl.RecipientBoxes))
	}

	// Then we simulate the propagation and retrieval by the client
	db, err := storage.InitLevelDb(dbPath + "2")
	if err != nil {
		t.Fatal(err)
	}

	enc2 := Init(
		db,
		[]string{"testdata/rcpt1.pub"},
		[]string{"testdata/rcpt1"},
		pi,
		client)

	var digest2 []byte
	digest2, err = enc2.StorePayload(propagatedPl)

	if !bytes.Equal(digest, digest2) {
		t.Errorf("Local and propgated digests should be equal, local: %v, propagated: %v\n",
			digest, digest2)
	}

	var returned2 []byte
	to := (*rcpt1)[:]
	returned2, err = enc2.Retrieve(&digest2, &to)

	if !bytes.Equal(message, returned2) {
		t.Errorf(
			"Retrieved message is not the same as original:\n" +
				"Original: %v\nRetrieved: %v",
			message, returned)
	}
}

func TestStoreAndRetrieveSelf(t *testing.T) {
	dbPath, err := ioutil.TempDir("", "TestStoreAndRetrieveSelf")

	if err != nil {
		t.Fatal(err)
	} else {
		defer os.RemoveAll(dbPath)
	}

	enc := initDefaultEnclave(t, dbPath)

	digest, err := enc.Store(&message, []byte{}, [][]byte{})
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
	dbPath, err := ioutil.TempDir("", "TestStoreNotAuthorised")

	if err != nil {
		t.Fatal(err)
	} else {
		defer os.RemoveAll(dbPath)
	}

	enc := initDefaultEnclave(t, dbPath)

	pubKeys, err := loadPubKeys([]string{"testdata/rcpt1.pub"})
	if err != nil {
		t.Fatal(err)
	}
	rcpt1 := pubKeys[0]

	_, err = enc.Store(&message, (*rcpt1)[:], [][]byte{(*rcpt1)[:]})
	if err == nil {
		t.Error("SecureEnclave is not authorised to store messages")
	}
}

func TestRetrieveInvalid(t *testing.T) {
	dbPath, err := ioutil.TempDir("", "TestRetrieveInvalid")

	if err != nil {
		t.Fatal(err)
	} else {
		defer os.RemoveAll(dbPath)
	}

	enc := initDefaultEnclave(t, dbPath)

	digest := []byte("invalid")
	_, err = enc.Retrieve(&digest, nil)
	if err == nil {
		t.Error("Invalid digest requested")
	}
}

func TestRetrieveNotAuthorised(t *testing.T) {
	// If you know the source enclave of the message, you can retrieve passing in any value you
	// want in the to field. This may not be appropriate.
	// TODO: Confirm if we want to do this
	dbPath, err := ioutil.TempDir("", "TestRetrieveNotAuthorised")

	if err != nil {
		t.Fatal(err)
	} else {
		defer os.RemoveAll(dbPath)
	}

	enc := initDefaultEnclave(t, dbPath)

	pubKeys, err := loadPubKeys([]string{"testdata/rcpt1.pub", "testdata/rcpt2.pub"})
	if err != nil {
		t.Fatal(err)
	}
	rcpt1 := pubKeys[0]
	rcpt2 := pubKeys[1]

	var digest []byte
	digest, err = enc.Store(&message, []byte{}, [][]byte{(*rcpt1)[:]})
	if err != nil {
		t.Fatal(err)
	}

	var returned []byte
	to := (*rcpt2)[:]
	// we may want this to fail, as it won't work if the message didn't originate with us
	returned, err = enc.Retrieve(&digest, &to)

	if !bytes.Equal(message, returned) {
		t.Errorf(
			"Retrieved message is not the same as original:\n" +
				"Original: %v\nRetrieved: %v",
			message, returned)
	}
}

func TestDelete(t *testing.T) {
	dbPath, err := ioutil.TempDir("", "TestDelete")

	if err != nil {
		t.Fatal(err)
	} else {
		defer os.RemoveAll(dbPath)
	}

	enc := initDefaultEnclave(t, dbPath)

	digest, err := enc.Store(&message, []byte{}, [][]byte{})
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

	err = enc.Delete(&digest)
	if err != nil {
		t.Errorf("Unable to delete payload for key: %v\n", &digest)
	}

	_, err = enc.Retrieve(&digest, nil)
	if err == nil {
		t.Errorf("No error returned requesting invalid payload")
	}
}

func TestRetrieveFor(t *testing.T) {
	dbPath, err := ioutil.TempDir("", "TestRetrieveFor")

	if err != nil {
		t.Fatal(err)
	} else {
		defer os.RemoveAll(dbPath)
	}

	enc := initDefaultEnclave(t, dbPath)

	pubKeys, err := loadPubKeys([]string{"testdata/rcpt1.pub"})
	if err != nil {
		t.Fatal(err)
	}
	rcpt1 := (*pubKeys[0])[:]

	digest, err := enc.Store(&message, []byte{}, [][]byte{rcpt1})
	if err != nil {
		t.Fatal(err)
	}

	var returned *[]byte
	returned, err = enc.RetrieveFor(&digest, &rcpt1)

	epl := api.DecodePayload(*returned)

	if len(epl.RecipientBoxes) != 1 {
		t.Errorf("Retrieved record does not contain a single box, total: %d",
			len(epl.RecipientBoxes))
	}
}

func TestRetrieveForInvalid(t *testing.T) {
	dbPath, err := ioutil.TempDir("", "TestRetrieveFor")

	if err != nil {
		t.Fatal(err)
	} else {
		defer os.RemoveAll(dbPath)
	}

	enc := initDefaultEnclave(t, dbPath)

	var pubKeys []nacl.Key
	pubKeys, err = loadPubKeys([]string{"testdata/rcpt1.pub"})
	if err != nil {
		t.Fatal(err)
	}
	rcpt1 := (*pubKeys[0])[:]

	digest := []byte("Invalid")
	_, err = enc.RetrieveFor(&digest, &rcpt1)

	if err == nil {
		t.Error("No error returned requesting invalid payload")
	}
}

func TestRetrieveAllFor(t *testing.T) {
	dbPath, err := ioutil.TempDir("", "TestRetrieveAllFor")

	if err != nil {
		t.Fatal(err)
	} else {
		defer os.RemoveAll(dbPath)
	}

	mockClient := &MockClient{requests: [][]byte{}}
	var client utils.HttpClient
	client = mockClient

	pubKeys, err := loadPubKeys([]string{"testdata/rcpt1.pub"})
	if err != nil {
		t.Fatal(err)
	}
	rcpt1 := pubKeys[0]

	pi := api.CreatePartyInfo(
		"http://localhost:8000",
		[]string{"http://localhost:8001"},
		[]nacl.Key{rcpt1},
		client)

	enc := initEnclave(t, dbPath, pi, client)

	_, err = enc.Store(&message, []byte{}, [][]byte{(*rcpt1)[:]})
	if err != nil {
		t.Fatal(err)
	}

	message2 := []byte("Another message")
	_, err = enc.Store(&message2, []byte{}, [][]byte{(*rcpt1)[:]})
	if err != nil {
		t.Fatal(err)
	}

	rcpt1Key := (*rcpt1)[:]
	err = enc.RetrieveAllFor(&rcpt1Key)
	if err != nil {
		t.Fatal(err)
	}

	// we need to wait for the replay go-routines to complete
	time.Sleep(1 * time.Millisecond)
	if mockClient.reqCount() != 4 {
		t.Errorf("Four requests should have been captured, actual: %d\n",
			len(mockClient.requests))
	}
}

func TestDoKeyGeneration(t *testing.T) {
	dbPath, err := ioutil.TempDir("", "TestDoKeyGeneration")

	if err != nil {
		t.Fatal(err)
	} else {
		defer os.RemoveAll(dbPath)
	}

	keyFiles := path.Join(dbPath, "testKey")
	err = DoKeyGeneration(keyFiles)

	if err != nil {
		t.Fatal(err)
	}

	_, err = loadPubKeys([]string{keyFiles + ".pub"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = loadPrivKeys([]string{keyFiles + ".key"})
	if err != nil {
		t.Fatal(err)
	}
}
