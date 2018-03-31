package storage

type DataStore interface {
	Write(key *[]byte, value *[]byte) error
	Read(key *[]byte) (*[]byte, error)
	ReadAll(f func(key, value *[]byte)) error
	Delete(key *[]byte) error
}
