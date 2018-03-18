package api

import (
	"encoding/binary"
	"github.com/kevinburke/nacl"
	"bytes"
	"net/http"
	"io/ioutil"
)

type EncryptedPayload struct {
	Sender         nacl.Key
	CipherText     []byte
	Nonce          nacl.Nonce
	RecipientBoxes [][]byte
	RecipientNonce nacl.Nonce
}

type PartyInfo struct {
	Url string
	// public key -> URL
	Recipients map[string]string
	Parties map[string]bool  // URLs
}

func EncodePayload(ep EncryptedPayload) []byte {
	// constant fields are 216 bytes
	encoded := make([]byte, 512)

	offset := 0
	var bytes int

	encoded, bytes = writeSlice((*ep.Sender)[:], encoded, offset)
	offset += bytes

	encoded, bytes = writeSlice(ep.CipherText, encoded, offset)
	offset += bytes

	encoded, bytes = writeSlice((*ep.Nonce)[:], encoded, offset)
	offset += bytes

	encoded, bytes = writeSliceOfSlice(ep.RecipientBoxes, encoded, offset)
	offset += bytes

	encoded, bytes = writeSlice((*ep.RecipientNonce)[:], encoded, offset)
	offset += bytes

	return encoded
}

func DecodePayload(encoded []byte) (EncryptedPayload, error) {

	ep := EncryptedPayload{
		Sender: new([nacl.KeySize]byte),
		Nonce: new([nacl.NonceSize]byte),
		RecipientNonce: new([nacl.NonceSize]byte),
	}

	pos := 0
	length := readInt(encoded[0:8])
	pos += 8
	copy((*ep.Sender)[:], encoded[pos:pos + length])
	pos += length

	length = readInt(encoded[pos:pos + 8])
	pos += 8
	ep.CipherText = encoded[pos:pos + length]
	pos += length

	length = readInt(encoded[pos:pos + 8])
	pos += 8
	copy((*ep.Nonce)[:], encoded[pos:pos + length])
	pos += length

	arraySize := readInt(encoded[pos:pos + 8])
	pos += 8
	ep.RecipientBoxes = make([][]byte, arraySize)
	for i := 0; i < arraySize; i++ {
		length = readInt(encoded[pos:pos + 8])
		pos += 8
		ep.RecipientBoxes[i] = append(
			ep.RecipientBoxes[i], encoded[pos:pos + length]...)
		pos += length
	}

	length = readInt(encoded[pos:pos + 8])
	pos += 8
	copy((*ep.RecipientNonce)[:], encoded[pos:pos + length])
	pos += length

	return ep, nil
}

func writeInt(v int, dest []byte, offset int) {
	binary.BigEndian.PutUint64(dest[offset:], uint64(v))
}

func readInt(src []byte) int {
	return int(binary.BigEndian.Uint64(src))
}

func writeSlice(src []byte, dest []byte, offset int) ([]byte, int) {
	length := len(src)

	writeInt(length, dest, offset)
	copy(dest[offset + 8:], src)
	return dest, 8 + length
}

func writeSliceOfSlice(src [][]byte, dest []byte, offset int) ([]byte, int) {

	length := len(src)
	writeInt(length, dest, offset)
	totalBytes := 8
	currOffset := offset + 8

	for _, b := range src {
		var bytes int
		dest, bytes = writeSlice(b, dest, currOffset)
		currOffset += bytes
		totalBytes += bytes
	}

	return dest, totalBytes
}

func Push(epl EncryptedPayload, url string) (string, error) {

	encodedPl := EncodePayload(epl)

	resp, err := http.Post(
		url + "/push", "application/octet-stream", bytes.NewReader(encodedPl))
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		return "", err
	}

	return string(body), nil
}