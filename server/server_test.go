package server

import (
	"testing"
	"net/http"
	"net/http/httptest"
)

func TestUpcheck(t *testing.T) {
	req, err := http.NewRequest("GET", "/upcheck", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	tm := TransactionManager{}

	handler := http.HandlerFunc(tm.upcheck)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	if rr.Body.String() != upCheckResponse {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), upCheckResponse)
	}
}

func TestSendAndReceive(t *testing.T) {

}

func TestPushAndReceive(t *testing.T) {

}

func TestDelete(t *testing.T) {

}

func TestResendIndividual(t *testing.T) {

}

func TestResendAll(t *testing.T) {

}

func TestPartyInfo(t *testing.T) {

}
