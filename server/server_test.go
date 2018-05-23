package server

import (
	"testing"
	"net/http"
	"net/http/httptest"
	"github.com/blk-io/crux/api"
	"encoding/json"
	"encoding/base64"
	"bytes"
	"reflect"
	"github.com/kevinburke/nacl"
	"github.com/blk-io/crux/enclave"
	"github.com/blk-io/crux/storage"
	"path"
	"io/ioutil"
)

const sender = "BULeR8JyUWhiuuCMU/HLA0Q5pzkYT+cHII3ZKBey3Bo="
const receiver = "QfeDAys9MPDs2XHExtc84jKGHxZg/aj52DTh0vtA3Xc="

var payload = []byte("payload")
var encodedPayload = base64.StdEncoding.EncodeToString(payload)

type MockEnclave struct {}

func (s* MockEnclave) Store(message *[]byte, sender []byte, recipients [][]byte) ([]byte, error) {
	return *message, nil
}

func (s* MockEnclave) StorePayload(encoded []byte) ([]byte, error) {
	return encoded, nil
}

func (s* MockEnclave) Retrieve(digestHash *[]byte, to *[]byte) ([]byte, error) {
	return *digestHash, nil
}

func (s* MockEnclave) RetrieveDefault(digestHash *[]byte) ([]byte, error) {
	return *digestHash, nil
}

func (s* MockEnclave) RetrieveFor(digestHash *[]byte, reqRecipient *[]byte) (*[]byte, error) {
	return digestHash, nil
}

func (s* MockEnclave) RetrieveAllFor(reqRecipient *[]byte) error {
	return nil
}

func (s* MockEnclave) Delete(digestHash *[]byte) error {
	return nil
}

func (s* MockEnclave) UpdatePartyInfo(encoded []byte) {}

func (s* MockEnclave) GetEncodedPartyInfo() []byte {
	return payload
}

func TestUpcheck(t *testing.T) {
	tm := TransactionManager{}
	runSimpleGetRequest(t, upCheck, upCheckResponse, tm.upcheck)
}

func TestVersion(t *testing.T) {
	tm := TransactionManager{}
	runSimpleGetRequest(t, version, apiVersion, tm.version)
}

func runSimpleGetRequest(t *testing.T, url, response string, handlerFunc http.HandlerFunc) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(handlerFunc)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v\n",
			status, http.StatusOK)
	}

	if rr.Body.String() != response {
		t.Errorf("handler returned unexpected body: got %v want %v\n",
			rr.Body.String(), upCheckResponse)
	}
}

func TestSend(t *testing.T) {
	sendReqs := []api.SendRequest{
		{
			Payload: encodedPayload,
			From: sender,
			To: []string{receiver},
		},
		{
			Payload: encodedPayload,
			To: []string{},
		},
		{
			Payload: encodedPayload,
		},
	}

	response := api.SendResponse{}
	expected := api.SendResponse{Key: encodedPayload}

	tm := TransactionManager{Enclave: &MockEnclave{}}

	for _, sendReq := range sendReqs {
		runJsonHandlerTest(t, &sendReq, &response, &expected, send, tm.send)
	}
}

func TestSendRaw(t *testing.T) {
	tm := TransactionManager{Enclave: &MockEnclave{}}

	headers := make(http.Header)
	headers[hFrom] = []string{sender}
	headers[hTo] = []string{receiver}

	// Uncomment the below for Quorum v2.0.2 onwards
	//runRawHandlerTest(t, headers, payload, []byte(encodedPayload), sendRaw, tm.sendRaw)
	runRawHandlerTest(t, headers, payload, payload, sendRaw, tm.sendRaw)
}

func TestReceive(t *testing.T) {

	receiveReqs := []api.ReceiveRequest{
		{
			Key: encodedPayload,
			To: receiver,
		},
	}

	response := api.ReceiveResponse{}
	expected := api.ReceiveResponse{Payload: encodedPayload}

	tm := TransactionManager{Enclave: &MockEnclave{}}

	for _, receiveReq := range receiveReqs {
		runJsonHandlerTest(t, &receiveReq, &response, &expected, receive, tm.receive)
	}
}

func TestReceivedRaw(t *testing.T) {
	tm := TransactionManager{Enclave: &MockEnclave{}}

	headers := make(http.Header)
	headers[hKey] = []string{encodedPayload}
	headers[hTo] = []string{receiver}

	runRawHandlerTest(t, headers, payload, payload, receiveRaw, tm.receiveRaw)
}

func TestNilKeyReceivedRaw(t *testing.T) {
	tm := TransactionManager{Enclave: &MockEnclave{}}

	headers := make(http.Header)
	headers[hKey] = []string{""}
	headers[hTo] = []string{receiver}

	runFailingRawHandlerTest(t, headers, payload, payload, receiveRaw, tm.receiveRaw)
}

func TestPush(t *testing.T) {

	epl := api.EncryptedPayload{
		Sender: nacl.NewKey(),
		CipherText: []byte(payload),
		Nonce: nacl.NewNonce(),
		RecipientBoxes: [][]byte{[]byte(payload)},
		RecipientNonce: nacl.NewNonce(),
	}
	var recipients [][]byte

	encoded := api.EncodePayloadWithRecipients(epl, recipients)

	req, err := http.NewRequest("POST", push, bytes.NewBuffer(encoded))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	tm := TransactionManager{Enclave: &MockEnclave{}}

	handler := http.HandlerFunc(tm.push)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v\n",
			status, http.StatusOK)
	}

	if !bytes.Equal(rr.Body.Bytes(), encoded) {
		t.Errorf("handler returned unexpected body: got %v wanted %v\n",
			rr.Body.String(), encoded)
	}
}

func TestDelete(t *testing.T) {
	sendReq := api.DeleteRequest{
		Key: encodedPayload,
	}

	var response, expected interface{}

	tm := TransactionManager{Enclave: &MockEnclave{}}

	runJsonHandlerTest(t, &sendReq, &response, &expected, delete, tm.delete)
}

func runJsonHandlerTest(
	t *testing.T,
	request, response, expected interface{},
	url string,
	handlerFunc http.HandlerFunc) {

	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}

	var req *http.Request
	req, err = http.NewRequest("POST", url, bytes.NewBuffer(encoded))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(handlerFunc)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	body := rr.Body.Bytes()
	if len(body) > 0 {
		err = json.Unmarshal(body, &response)
		if err != nil {
			t.Error(err)
		}
	}

	if !reflect.DeepEqual(response, expected) {
		t.Errorf("handler returned unexpected response: %v, expected: %v\n", response, expected)
	}
}
func runFailingRawHandlerTest(
	t *testing.T,
	headers http.Header,
	payload, expected []byte,
	url string,
	handlerFunc http.HandlerFunc) {

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		t.Fatal(err)
	}

	for k, v := range headers {
		req.Header.Set(k, v[0])
	}

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(handlerFunc)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status == http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func runRawHandlerTest(
	t *testing.T,
	headers http.Header,
	payload, expected []byte,
	url string,
	handlerFunc http.HandlerFunc) {

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		t.Fatal(err)
	}

	for k, v := range headers {
		req.Header.Set(k, v[0])
	}

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(handlerFunc)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	response := rr.Body.Bytes()

	if !reflect.DeepEqual(response, expected) {
		t.Errorf("handler returned unexpected response: %v, expected: %v\n", response, expected)
	}
}


func TestResendIndividual(t *testing.T) {
	resendReq := api.ResendRequest{
		Type:       "individual",
		PublicKey:  sender,
		Key:        encodedPayload,
	}

	body := runResendTest(t, resendReq)

	if !bytes.Equal(body, payload) {
		t.Errorf("handler returned unexpected body: got %v wanted %v\n",
			body, payload)
	}
}

func TestResendAll(t *testing.T) {
	resendReq := api.ResendRequest{
		Type:       "all",
		PublicKey:  sender,
	}

	body := runResendTest(t, resendReq)

	if len(body) != 0 {
		t.Errorf("handler returned unexpected body, it should be empty, instead received: %v\n",
			body)
	}
}

func runResendTest(t *testing.T, resendReq api.ResendRequest) []byte {
	encoded, err := json.Marshal(resendReq)
	if err != nil {
		t.Error(err)
	}

	var req *http.Request
	req, err = http.NewRequest("POST", push, bytes.NewBuffer(encoded))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	tm := TransactionManager{Enclave: &MockEnclave{}}

	handler := http.HandlerFunc(tm.resend)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v\n",
			status, http.StatusOK)
	}

	return rr.Body.Bytes()
}

func TestPartyInfo(t *testing.T) {

	partyInfos := []api.PartyInfo{
		api.CreatePartyInfo(
			"http://localhost:8000",
			[]string{"http://localhost:8001"},
			[]nacl.Key{nacl.NewKey()},
			http.DefaultClient),

		api.InitPartyInfo(
			"http://localhost:8000",
			[]string{"http://localhost:8001"},
			http.DefaultClient),
	}

	for _, pi := range partyInfos {
		testRunPartyInfo(t, pi)
	}
}

func testRunPartyInfo(t *testing.T, pi api.PartyInfo) {
	encoded := api.EncodePartyInfo(pi)

	req, err := http.NewRequest("POST", push, bytes.NewBuffer(encoded))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	tm := TransactionManager{Enclave: &MockEnclave{}}

	handler := http.HandlerFunc(tm.partyInfo)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v\n",
			status, http.StatusOK)
	}

	if !bytes.Equal(rr.Body.Bytes(), payload) {
		t.Errorf("handler returned unexpected body: got %v wanted %v\n",
			rr.Body.Bytes(), payload)
	}
}

func TestInit(t *testing.T) {
	dbPath, err := ioutil.TempDir("", "TestInit")
	if err != nil {
		t.Error(err)
	}
	db, err := storage.InitLevelDb(dbPath)
	if err != nil {
		t.Errorf("Error starting server: %v\n", err)
	}
	pubKeyFiles := []string{"key.pub"}
	privKeyFiles := []string{"key"}

	for i, keyFile := range privKeyFiles {
		privKeyFiles[i] = path.Join("../enclave/testdata", keyFile)
	}

	for i, keyFile := range pubKeyFiles {
		pubKeyFiles[i] = path.Join("../enclave/testdata", keyFile)
	}

	key := []nacl.Key{nacl.NewKey()}

	pi := api.CreatePartyInfo(
		"http://localhost:9000",
		[]string{"http://localhost:9001"},
		key,
		http.DefaultClient)

	enc := enclave.Init(db, pubKeyFiles, privKeyFiles, pi, http.DefaultClient)

	ipcPath, err := ioutil.TempDir("", "TestInitIpc")
	if err != nil {
		t.Error(err)
	}
	tm, err := Init(enc, 9001, ipcPath)
	if err != nil {
		t.Errorf("Error starting server: %v\n", err)
	}
	runSimpleGetRequest(t, upCheck, upCheckResponse, tm.upcheck)
}
