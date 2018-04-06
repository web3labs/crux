package enclave

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
	"gitlab.com/blk-io/crux/storage"
	"gitlab.com/blk-io/crux/api"
	"net/http"
	"gitlab.com/blk-io/crux/utils"
	"github.com/kevinburke/nacl"
)

var message = []byte("Test message")

type MockClient struct {
	requests [][]byte
}

func (c *MockClient) Do(req *http.Request) (*http.Response, error) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	c.requests = append(c.requests, body)
	respBody := ioutil.NopCloser(bytes.NewReader([]byte("")))
	return &http.Response{Body: respBody}, nil
}

func initEnclave(
	t *testing.T,
	dbPath string,
	pi api.PartyInfo,
	client utils.HttpClient) Enclave {
	db, err := storage.Init(dbPath)
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
	dbPath string) Enclave {

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

	pubKeys, err := loadPubKeys([]string{"testdata/rcpt1.pub", "testdata/rcpt2.pub"})
	rcpt1 := pubKeys[1]

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

	// Verify payload propagation too
	if len(mockClient.requests) != 1 {
		t.Errorf("Only one request should have been captured, actual: %d\n",
			len(mockClient.requests))
	}

	epl, recipients := api.DecodePayloadWithRecipients(mockClient.requests[0])

	if len(recipients) != 0 {
		t.Errorf("Recipients should be empty in data sent to other nodes, actual size: %d\n",
			len(recipients))
	}

	if len(epl.RecipientBoxes) != 1 {
		t.Errorf("There should only be one recipient box present, actual %d\n",
			len(epl.RecipientBoxes))
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
	rcpt1 := pubKeys[0]

	_, err = enc.Store(&message, (*rcpt1)[:], [][]byte{(*rcpt1)[:]})
	if err == nil {
		t.Error("Enclave is not authorised to store messages")
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
