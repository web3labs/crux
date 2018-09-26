package main

import (
	"github.com/blk-io/crux/api"
	"github.com/blk-io/crux/config"
	"github.com/blk-io/crux/enclave"
	"github.com/blk-io/crux/server"
	"github.com/blk-io/crux/storage"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

func main() {

	config.InitFlags()

	args := os.Args
	if len(args) == 1 {
		exit()
	}

	for _, arg := range args[1:] {
		if strings.Contains(arg, ".conf") {
			err := config.LoadConfig(arg)
			if err != nil {
				log.Fatalln(err)
			}
			break
		}
	}
	config.ParseCommandLine()

	verbosity := 1
	if config.GetInt(config.Verbosity) > config.GetInt(config.VerbosityShorthand) {
		verbosity = config.GetInt(config.Verbosity)
	} else {
		verbosity = config.GetInt(config.VerbosityShorthand)
	}

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
		log.Fatalf("Unable to initialise storage, error: %v", err)
	}
	defer db.Close()

	allOtherNodes := config.GetString(config.OtherNodes)
	otherNodes := strings.Split(allOtherNodes, ",")
	url := config.GetString(config.Url)
	if url == "" {
		log.Fatalln("URL must be specified")
	}
	port := config.GetInt(config.Port)
	if port < 0 {
		log.Fatalln("Port must be specified")
	}
	httpClient := &http.Client{
		Timeout: time.Second * 10,
	}
	grpc := config.GetBool(config.UseGRPC)

	pi := api.InitPartyInfo(url, otherNodes, httpClient, grpc)

	privKeys := config.GetString(config.PrivateKeys)
	pubKeys := config.GetString(config.PublicKeys)
	pubKeyFiles := strings.Split(pubKeys, ",")
	privKeyFiles := strings.Split(privKeys, ",")

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

	enc := enclave.Init(db, pubKeyFiles, privKeyFiles, pi, http.DefaultClient, grpc)

	pi.RegisterPublicKeys(enc.PubKeys)

	tls := config.GetBool(config.Tls)
	var tlsCertFile, tlsKeyFile string
	if tls {
		servCert := config.GetString(config.TlsServerCert)
		servKey := config.GetString(config.TlsServerKey)

		if (len(servCert) != len(servKey)) || (len(servCert) <= 0) {
			log.Fatalf("Please provide server certificate and key for TLS %s %s %d ", servKey, servCert, len(servCert))
		}

		tlsCertFile = path.Join(workDir, servCert)
		tlsKeyFile = path.Join(workDir, servKey)
	}
	grpcJsonport := config.GetInt(config.GrpcJsonPort)
	_, err = server.Init(enc, port, ipcPath, grpc, grpcJsonport, tls, tlsCertFile, tlsKeyFile)
	if err != nil {
		log.Fatalf("Error starting server: %v\n", err)
	}

	pi.PollPartyInfo()

	select {}
}

func exit() {
	config.Usage()
	os.Exit(1)
}
