package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"gitlab.com/blk-io/crux/config"
	"gitlab.com/blk-io/crux/enclave"
	"gitlab.com/blk-io/crux/server"
	"gitlab.com/blk-io/crux/storage"
	"gitlab.com/blk-io/crux/api"
	"net"
	"path/filepath"
	"strings"
	"github.com/kevinburke/nacl"
	"encoding/base64"
	"io/ioutil"
	"encoding/json"
	"github.com/kevinburke/nacl/box"
	"crypto/rand"
)

func main() {

	config.InitFlags()
	for _, arg := range os.Args {
		if strings.Contains(arg, ".conf") {
			err := config.LoadConfig(os.Args[0])
			if err != nil {
				log.Fatal(err)
			}
			break
		}
	}
	config.ParseCommandLine()

	generateKeys := config.GetString(config.GenerateKeys)
	if generateKeys != "" {
		doKeyGeneration(generateKeys)
		os.Exit(0)
	}

	// "/Users/Conor/code/go/blk-io/tmp/crux.db"

	db, err := storage.Init(config.GetString(config.Storage))
	if err != nil {
		log.Fatal(err)
	}

	// constellation-node --url=https://127.0.0.7:9007/ --port=9007 --workdir=qdata/c7 --socket=tm.ipc --publickeys=tm.pub --privatekeys=tm.key --othernodes=https://127.0.0.1:9001/ >> qdata/logs/constellation7.log 2>&1 &

	otherNodes := config.GetStringSlice(config.OtherNodes)
	pi := loadPartyInfo(otherNodes)

	privKeyFiles := config.GetStringSlice(config.PrivateKeys)
	pubKeyFiles := config.GetStringSlice(config.PublicKeys)

	if len(privKeyFiles) != len(pubKeyFiles) {
		log.Fatal("Private keys provided must have corresponding public keys")
	}

	// BULeR8JyUWhiuuCMU/HLA0Q5pzkYT+cHII3ZKBey3Bo=
	pubKeys, err := loadPubKeys(pubKeyFiles)
	if err != nil {
		log.Fatalf("Unable to load public key files: %s, error: %q", pubKeyFiles, err)
	}

	// {"data":{"bytes":"Wl+xSyXVuuqzpvznOS7dOobhcn4C5auxkFRi7yLtgtA="},"type":"unlocked"}
	privKeys, err := loadPrivKeys(pubKeyFiles)
	if err != nil {
		log.Fatalf("Unable to load private key files: %s, error: %q", pubKeyFiles, err)
	}

	enc := enclave.Enclave{
		Db : db,
		PubKeys: pubKeys,
		PrivKeys: privKeys,
		PartyInfo: pi,
	}
	enc.Init()

	tm := server.TransactionManager{Enclave : enc}

	httpServer := http.NewServeMux()
	httpServer.HandleFunc("/upcheck", tm.Upcheck)
	httpServer.HandleFunc("/push", tm.Push)
	httpServer.HandleFunc("/resend", tm.Resend)
	httpServer.HandleFunc("/partyinfo", tm.PartyInfo)

	port := config.GetInt(config.Port)
	go log.Fatal(http.ListenAndServe("localhost:" + strconv.Itoa(port), httpServer))

	// Restricted to IPC
	ipcServer := http.NewServeMux()
	ipcServer.HandleFunc("/send", tm.Send)
	ipcServer.HandleFunc("/receive", tm.Receive)
	ipcServer.HandleFunc("/delete", tm.Delete)

	var ipc net.Listener
	ipc, err = createIpcSocket("")
	go log.Fatal(http.Serve(ipc, ipcServer))

	// TODO: Send initial request for partyInfo
}

func loadPubKeys(pubKeyFiles []string) ([]nacl.Key, error) {
	return loadKeys(
		pubKeyFiles,
		func(s string) (string, error) {
			src, err := ioutil.ReadFile(s)
			if err != nil {
				return "", err
			}
			return string(src), nil
		})
}

func loadPrivKeys(privKeyFiles []string) ([]nacl.Key, error) {
	return loadKeys(
		privKeyFiles,
		func(s string) (string, error) {
			var privateKey api.PrivateKey
			src, err := ioutil.ReadFile(s)
			if err != nil {
				return "", err
			}
			err = json.Unmarshal(src, privateKey)
			if err != nil {
				return "", err
			}

			return privateKey.Data.Bytes, nil
		})
}

func loadKeys(
	keyFiles []string, f func(string) (string, error)) ([]nacl.Key, error) {
	keys := make([]nacl.Key, len(keyFiles))

	for i, keyFile := range keyFiles {
		data, err := f(keyFile)
		if err != nil {
			return nil, err
		}
		var key nacl.Key
		key, err = loadBase64Key(
			strings.TrimSuffix(data, "\n"))
		if err != nil {
			return nil, err
		}
		keys[i] = key
	}

	return keys, nil
}

func doKeyGeneration(keyFile string) {
	pubKey, privKey, err := box.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatal("Error creating keys")
	}
	err = createDirForFile(keyFile)
	if err != nil {
		log.Fatalf("Invalid destination specified: %s", filepath.Dir(keyFile))
	}

	b64PubKey := base64.StdEncoding.EncodeToString((*pubKey)[:])
	b64PrivKey := base64.StdEncoding.EncodeToString((*privKey)[:])

	err = ioutil.WriteFile(keyFile + ".pub", []byte(b64PubKey), 0600)
	if err != nil {
		log.Fatalf("Unable to write public key: %s, error: %q", keyFile, err)
	}

	jsonKey := api.PrivateKey{
		Type: "unlocked",
		Data: api.PrivateKeyBytes{
			Bytes: b64PrivKey,
		},
	}

	var encoded []byte
	encoded, err = json.Marshal(jsonKey)
	if err != nil {
		log.Fatalf("Unable to encode private key: %q, error: %q", jsonKey, err)
	}

	err = ioutil.WriteFile(keyFile, encoded, 0600)
	if err != nil {
		log.Fatalf("Unable to write private key: %s, error: %q", keyFile, err)
	}
}

func loadPartyInfo(otherNodes []string) api.PartyInfo {
	parties := make(map[string]bool)
	for _, node := range otherNodes {
		parties[node] = true
	}

	return api.PartyInfo{
		Url: config.GetString(config.Url),
		Recipients: make(map[string]string),
		Parties: parties,
	}
}

func loadBase64Key(key string) (nacl.Key, error) {
	src, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return nil, err
	}

	return enclave.ToKey(src)
}

func createIpcSocket(path string) (net.Listener, error) {
	err := createDirForFile(path)
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

func createDirForFile(path string) error {
	return os.MkdirAll(filepath.Dir(path), os.FileMode(0755))
}
