package api

import (
	"testing"
)

func TestPush(t *testing.T) {

}

func TestEncodeInitPartyInfo(t *testing.T) {

	pi := InitPartyInfo("https://127.0.0.1:9001/",
		[]string{"http://127.0.0.1:9002"}, nil)

	runEncodePartyInfoTest(t, pi)
}