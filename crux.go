package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"gitlab.com/eea/crux/config"
	"gitlab.com/eea/crux/enclave"
	"gitlab.com/eea/crux/server"
	"gitlab.com/eea/crux/storage"
	"gitlab.com/eea/crux/api"
)

func main() {

	if len(os.Args) != 1 {
		log.Fatal("Configuration file must be specified")
	}

	err := config.LoadConfig(os.Args[0])

	if err != nil {
		log.Fatal(err)
	}

	// "/Users/Conor/code/go/blk-io/tmp/crux.db"

	db, err := storage.Init(config.GetString(config.Storage))
	if err != nil {
		log.Fatal(err)
	}

	// TODO: Support command line args
	// constellation-node --url=https://127.0.0.7:9007/ --port=9007 --workdir=qdata/c7 --socket=tm.ipc --publickeys=tm.pub --privatekeys=tm.key --othernodes=https://127.0.0.1:9001/ >> qdata/logs/constellation7.log 2>&1 &

	otherNodes := config.GetStringSlice(config.OtherNodes)
	parties := make(map[string]bool)
	for _, node := range otherNodes {
		parties[node] = true
	}

	pi := api.PartyInfo{
		Url: config.GetString(config.Url),
		Recipients: make(map[string]string),
		Parties: parties,
	}

	// TODO: Populate public & private keys
	enc := enclave.Enclave{
		Db : db,
		PubKeys: nil,
		PrivKeys: nil,
		PartyInfo: pi,
	}

	// TODO: Read key from configuration & add command line tool to generate
	tm := server.TransactionManager{Enclave : enc}

	http.HandleFunc("/upcheck", tm.Upcheck)
	http.HandleFunc("/push", tm.Push)
	http.HandleFunc("/resend", tm.Resend)
	http.HandleFunc("/partyinfo", tm.PartyInfo)

	// TODO: Restrict to IPC
	port := config.GetInt(config.Port)
	http.HandleFunc("/send", tm.Send)
	http.HandleFunc("/receive", tm.Receive)
	http.HandleFunc("/delete", tm.Delete)
	log.Fatal(http.ListenAndServe("localhost:" + strconv.Itoa(port), nil))

	// TODO: Add support for propagation methods
	// Propagate party info

	// TODO: Add support for replay
}
