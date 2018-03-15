package main

import (
	"log"
	"net/http"
	"github.com/blk-io/crux/config"
	"github.com/blk-io/crux/enclave"
	"github.com/blk-io/crux/server"
	"github.com/blk-io/crux/storage"
	"os"
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

	// constellation-node --url=https://127.0.0.7:9007/ --port=9007 --workdir=qdata/c7 --socket=tm.ipc --publickeys=tm.pub --privatekeys=tm.key --othernodes=https://127.0.0.1:9001/ >> qdata/logs/constellation7.log 2>&1 &

	enc := enclave.Enclave{Db : db}



	// TODO: Read key from configuration & add command line tool to generate
	tm := server.TransactionManager{Key : enclave.NewKey(), Enclave : enc}

	http.HandleFunc("/upcheck", tm.Upcheck)

	// TODO: Restrict to IPC
	http.HandleFunc("/send", tm.Send)
	http.HandleFunc("/receive", tm.Receive)
	log.Fatal(http.ListenAndServe("localhost:8000", nil))

	// TODO: Add support for propagation methods
}
