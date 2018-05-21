package utils

import (
	"testing"
)

func TestCreateIpcSocket(t *testing.T) {
	listener, err := CreateIpcSocket("data/crux.db")

	if err != nil{
		t.Error(err)
	}

	if listener == nil {
		t.Errorf("Listener not initialised")
	}
}
