package storage

type DataStore interface {
	Write(key *[]byte, value *[]byte)
	Read(key *[]byte) *[]byte
	Delete(key *[]byte)
}
