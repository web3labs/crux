package storage

import (
	"github.com/syndtr/goleveldb/leveldb"
)

type levelDb struct {
	dbPath string
	conn   *leveldb.DB
}

func InitLevelDb(dbPath string) (*levelDb, error) {
	db := new(levelDb)
	db.dbPath = dbPath
	var err error
	db.conn, err = leveldb.OpenFile(dbPath, nil)
	return db, err
}

func (db *levelDb) Write(key *[]byte, value *[]byte) error {
	return db.conn.Put(*key, *value, nil)
}

func (db *levelDb) Read(key *[]byte) (*[]byte, error) {
	value, err := db.conn.Get(*key, nil)
	if err == nil {
		return &value, err
	} else {
		return nil, err
	}
}

func (db *levelDb) ReadAll(f func(key, value *[]byte)) error {
	iter := db.conn.NewIterator(nil, nil)
	for iter.Next() {
		key, value := iter.Key(), iter.Value()
		f(&key, &value)
	}
	iter.Release()
	return iter.Error()
}

func (db *levelDb) Delete(key *[]byte) error {
	return db.conn.Delete(*key, nil)
}

func (db *levelDb) Close() error {
	return db.conn.Close()
}