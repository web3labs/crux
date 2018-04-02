package utils

import (
	"net"
	"os"
	"path/filepath"
)

func CreateIpcSocket(path string) (net.Listener, error) {
	err := CreateDirForFile(path)
	if err != nil {
		return nil, err
	}
	os.Remove(path)

	var listener net.Listener
	listener, err = net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	os.Chmod(path, 0600)

	return listener, nil
}

func CreateDirForFile(path string) error {
	return os.MkdirAll(filepath.Dir(path), os.FileMode(0755))
}
