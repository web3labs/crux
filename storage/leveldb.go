package storage

import "github.com/syndtr/goleveldb/leveldb"

type levelDb struct {
	dbPath string
	conn   *leveldb.DB
}

func Init(dbPath string) *levelDb {
	db := new(levelDb)
	db.dbPath = dbPath
	var err error
	db.conn, err = leveldb.OpenFile(dbPath, nil)
	if err != nil {
		panic(err)
	}
	return db
}

func (db *levelDb) Write(key *[]byte, value *[]byte) {
	err := db.conn.Put(*key, *value, nil)
	if err != nil {
		panic(err)
	}
}

func (db *levelDb) Read(key *[]byte) *[]byte {
	value, err := db.conn.Get(*key, nil)
	if err != nil {
		panic(err)
	}
	return &value
}

func (db *levelDb) Delete(key *[]byte) {
	db.conn.Delete(*key, nil)
}