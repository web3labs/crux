package api

import (
	"github.com/kevinburke/nacl"
	"encoding/binary"
	"github.com/blk-io/crux/utils"
)

func EncodePayload(ep EncryptedPayload) []byte {
	// constant fields are 216 bytes
	encoded := make([]byte, 512)

	offset := 0
	encoded, offset = writeSlice((*ep.Sender)[:], encoded, offset)
	encoded, offset = writeSlice(ep.CipherText, encoded, offset)
	encoded, offset = writeSlice((*ep.Nonce)[:], encoded, offset)
	encoded, offset = writeSliceOfSlice(ep.RecipientBoxes, encoded, offset)
	encoded, offset = writeSlice((*ep.RecipientNonce)[:], encoded, offset)

	return encoded
}

func DecodePayload(encoded []byte) EncryptedPayload {

	ep := EncryptedPayload{
		Sender: new([nacl.KeySize]byte),
		Nonce: new([nacl.NonceSize]byte),
		RecipientNonce: new([nacl.NonceSize]byte),
	}

	offset := 0
	offset = readSliceToArray(encoded, offset, (*ep.Sender)[:])
	ep.CipherText, offset = readSlice(encoded, offset)
	offset = readSliceToArray(encoded, offset, (*ep.Nonce)[:])
	ep.RecipientBoxes, offset = readSliceOfSlice(encoded, offset)
	offset = readSliceToArray(encoded, offset, (*ep.RecipientNonce)[:])

	return ep
}

func EncodePartyInfo(pi PartyInfo) []byte {

	encoded := make([]byte, 256)

	offset := 0

	encoded, offset = writeSlice([]byte(pi.Url), encoded, offset)
	encoded, offset = writeInt(len(pi.Recipients), encoded, offset)

	for recipient, url := range pi.Recipients {
		tuple := [][]byte{
			[]byte(recipient),
			[]byte(url),
		}
		encoded, offset = writeSliceOfSlice(tuple, encoded, offset)
	}

	parties := make([][]byte, len(pi.Parties))
	i := 0
	for party := range pi.Parties {
		parties[i] = []byte(party)
		i += 1
	}
	encoded, offset = writeSliceOfSlice(parties, encoded, offset)

	return encoded
}

func DecodePartyInfo(encoded []byte) PartyInfo {
	pi := PartyInfo{
		Recipients: make(map[string]string),
		Parties: make(map[string]bool),
	}

	offset := 0
	url, offset := readSlice(encoded, offset)
	pi.Url = string(url)

	size := readInt(encoded[offset:])
	offset += 8

	for i := 0; i < size; i++ {
		var kv [][]byte
		kv, offset = readSliceOfSlice(encoded, offset)
		pi.Recipients[string(kv[0])] = string(kv[1])
	}

	parties, offset := readSliceOfSlice(encoded, offset)
	for _, party := range parties {
		pi.Parties[string(party)] = true
	}

	return pi
}

func writeInt(v int, dest []byte, offset int) ([]byte, int) {
	dest = confirmCapacity(dest, offset, 8)
	binary.BigEndian.PutUint64(dest[offset:], uint64(v))
	return dest, offset + 8
}

func confirmCapacity(dest []byte, offset, required int) []byte {
	length := len(dest)
	if length - offset < required {
		var newLength int
		if required > length {
			newLength = utils.NextPowerOf2(required)
		} else {
			newLength = length
		}
		return append(dest, make([]byte, newLength)...)
	} else {
		return dest
	}
}

func readInt(src []byte) int {
	return int(binary.BigEndian.Uint64(src))
}

func writeSlice(src []byte, dest []byte, offset int) ([]byte, int) {
	length := len(src)
	dest, offset = writeInt(length, dest, offset)

	dest = confirmCapacity(dest, offset, length)
	copy(dest[offset:], src)
	return dest, offset + length
}

func readSliceToArray(src []byte, offset int, dest []byte) int {
	length := readInt(src[offset:offset + 8])
	offset += 8
	copy(dest, src[offset:offset + length])
	offset += length
	return offset
}

func readSlice(src []byte, offset int) ([]byte, int) {
	length := readInt(src[offset:offset + 8])
	offset += 8
	return src[offset:offset + length], offset + length
}

func writeSliceOfSlice(src [][]byte, dest []byte, offset int) ([]byte, int) {
	length := len(src)
	dest, offset = writeInt(length, dest, offset)

	for _, b := range src {
		dest, offset = writeSlice(b, dest, offset)
	}

	return dest, offset
}

func readSliceOfSlice(src []byte, offset int) ([][]byte, int) {
	arraySize := readInt(src[offset:offset + 8])
	offset += 8

	result := make([][]byte, arraySize)
	for i := 0; i < arraySize; i++ {
		length := readInt(src[offset:offset + 8])
		offset += 8
		result[i] = append(
			result[i], src[offset:offset + length]...)
		offset += length
	}
	return result, offset
}