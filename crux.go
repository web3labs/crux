package main

import (
	"os"
	"path"
	"net/http"
	"strings"
	"gitlab.com/blk-io/crux/api"
	"gitlab.com/blk-io/crux/config"
	"gitlab.com/blk-io/crux/enclave"
	"gitlab.com/blk-io/crux/server"
	"gitlab.com/blk-io/crux/storage"
	log "github.com/sirupsen/logrus"
)

func main() {

	config.InitFlags()

	args := os.Args
	if len(args) == 1 {
		exit()
	}

	for _, arg := range args[1:] {
		if strings.Contains(arg, ".conf") {
			err := config.LoadConfig(os.Args[0])
			if err != nil {
				log.Fatalln(err)
			}
			break
		}
	}
	config.ParseCommandLine()

	verbosity := config.GetInt(config.Verbosity)
	var level log.Level

	switch verbosity {
	case 0:
		level = log.FatalLevel
	case 1:
		level = log.WarnLevel
	case 2:
		level = log.InfoLevel
	case 3:
		level = log.DebugLevel
	}
	log.SetLevel(level)

	keyFile := config.GetString(config.GenerateKeys)
	if keyFile != "" {
		err := enclave.DoKeyGeneration(keyFile)
		if err != nil {
			log.Fatalln(err)
		}
		log.Printf("Key pair successfully written to %s", keyFile)
		os.Exit(0)
	}

	workDir := config.GetString(config.WorkDir)
	dbStorage := config.GetString(config.Storage)
	ipcFile := config.GetString(config.Socket)
	storagePath := path.Join(workDir, dbStorage)
	ipcPath := path.Join(workDir, ipcFile)

	var db storage.DataStore
	var err error
	if config.GetBool(config.BerkeleyDb) {
		db, err = storage.InitBerkeleyDb(storagePath)
	} else {
		db, err = storage.InitLevelDb(storagePath)
	}

	if err != nil {
		log.Fatalf("Unable to initialise storage, error: %v\n", err)
	}
	defer db.Close()

	otherNodes := config.GetStringSlice(config.OtherNodes)
	url := config.GetString(config.Url)
	if url == "" {
		log.Fatalln("URL must be specified")
	}

	pi := api.InitPartyInfo(url, otherNodes, http.DefaultClient)

	privKeyFiles := config.GetStringSlice(config.PrivateKeys)
	pubKeyFiles := config.GetStringSlice(config.PublicKeys)

	if len(privKeyFiles) != len(pubKeyFiles) {
		log.Fatalln("Private keys provided must have corresponding public keys")
	}

	if len(privKeyFiles) == 0 {
		log.Fatalln("Node key files must be provided")
	}

	for i, keyFile := range privKeyFiles {
		privKeyFiles[i] = path.Join(workDir, keyFile)
	}

	for i, keyFile := range pubKeyFiles {
		pubKeyFiles[i] = path.Join(workDir, keyFile)
	}

	enc := enclave.Init(db, pubKeyFiles, privKeyFiles, pi, http.DefaultClient)

	port := config.GetInt(config.Port)
	if port < 0 {
		log.Fatalln("Port must be specified")
	}

	_, err = server.Init(enc, port, ipcPath)
	if err != nil {
		log.Fatalf("Error starting server: %v\n", err)
	}

	pi.PollPartyInfo()
}

func exit() {
	config.Usage()
	os.Exit(1)
}