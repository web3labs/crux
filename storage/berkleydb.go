package storage

import (
	"encoding/base64"
	"github.com/jsimonetti/berkeleydb"
)

type berkleyDb struct {
	dbPath string
	conn   *berkeleydb.Db
}

func InitBerkeleyDb(dbPath string) (*berkleyDb, error) {
	bdb := &berkleyDb{dbPath: dbPath}

	db, err := berkeleydb.NewDB()
	if err != nil {
		return nil, err
	}

	err = db.Open(
		dbPath, berkeleydb.DbHash, berkeleydb.DbCreate)

	return bdb, err
}

func (db *berkleyDb) Write(key *[]byte, value *[]byte) error {

	b64Key := base64.StdEncoding.EncodeToString(*key)
	b64Value := base64.StdEncoding.EncodeToString(*value)

	err := db.conn.Put(b64Key, b64Value)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (db *berkleyDb) Read(key *[]byte) (*[]byte, error) {

	b64Key := base64.StdEncoding.EncodeToString(*key)

	value, err := db.conn.Get(b64Key)
	if err != nil {
		return nil, err
	}

	var decoded []byte
	decoded, err = base64.StdEncoding.DecodeString(value)
	return &decoded, err
}

func (db *berkleyDb) ReadAll(f func(key, value *[]byte)) error {
	iter, err := db.conn.Cursor()
	if err != nil {
		return err
	}

	var b64Key, b64Value string
	for {
		b64Key, b64Value, err = iter.GetNext()
		if err != nil {
			break
		}
		key, err := base64.StdEncoding.DecodeString(b64Key)
		if err != nil {
			break
		}
		value, err := base64.StdEncoding.DecodeString(b64Value)
		if err != nil {
			break
		}
		f(&key, &value)
	}

	return err
}

func (db *berkleyDb) Delete(key *[]byte) error {
	b64Key := base64.StdEncoding.EncodeToString(*key)
	return db.conn.Delete(b64Key)
}

func (db *berkleyDb) Close() error {
	return db.conn.Close()
}