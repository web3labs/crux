package utils

import (
	"testing"
	"io/ioutil"
)

func TestCreateIpcSocket(t *testing.T) {
	dbPath, err := ioutil.TempDir("", "TestCreateIpcSocket")
	if err != nil {
		t.Error(err)
	}

	listener, err := CreateIpcSocket(dbPath)

	if err != nil{
		t.Error(err)
	}

	if listener == nil {
		t.Errorf("Listener not initialised")
	}
}
