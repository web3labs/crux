package storage

type DataStore interface {
	Write(key *[]byte, value *[]byte) error
	Read(key *[]byte) (*[]byte, error)
	Delete(key *[]byte) error
}
