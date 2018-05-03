package storage

// DataStore is an interface that facilitates operations with an underlying persistent data store.
type DataStore interface {
	Write(key *[]byte, value *[]byte) error
	Read(key *[]byte) (*[]byte, error)
	ReadAll(f func(key, value *[]byte)) error
	Delete(key *[]byte) error
	Close() error
}
