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

	return encoded[:offset]
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

func EncodePayloadWithRecipients(ep EncryptedPayload, recipients [][]byte) []byte {
	encoded := make([][]byte, 2)

	encoded[0] = EncodePayload(ep)

	encodedRecipients := make([]byte, 256)
	encodedRecipients, recipientsLength := writeSliceOfSlice(recipients, encodedRecipients, 0)
	encoded[1] = encodedRecipients[:recipientsLength]

	encoded2, length := writeSliceOfSlice(encoded, make([]byte, 512), 0)
	return encoded2[:length]
}

func DecodePayloadWithRecipients(encoded []byte) (EncryptedPayload, [][]byte) {

	decoded, _ := readSliceOfSlice(encoded, 0)

	ep := DecodePayload(decoded[0])
	recipients, _ := readSliceOfSlice(decoded[1], 0)

	return ep, recipients
}

func EncodePartyInfo(pi PartyInfo) []byte {

	encoded := make([]byte, 256)

	offset := 0

	encoded, offset = writeSlice([]byte(pi.url), encoded, offset)
	encoded, offset = writeInt(len(pi.recipients), encoded, offset)

	for recipient, url := range pi.recipients {
		tuple := [][]byte{
			recipient[:],
			[]byte(url),
		}
		encoded, offset = writeSliceOfSlice(tuple, encoded, offset)
	}

	parties := make([][]byte, len(pi.parties))
	i := 0
	for party := range pi.parties {
		parties[i] = []byte(party)
		i += 1
	}
	encoded, offset = writeSliceOfSlice(parties, encoded, offset)

	return encoded
}

func DecodePartyInfo(encoded []byte) (PartyInfo, error) {
	pi := PartyInfo{
		recipients: make(map[[nacl.KeySize]byte]string),
		parties:    make(map[string]bool),
	}

	url, offset := readSlice(encoded, 0)
	pi.url = string(url)

	var size int
	size, offset = readInt(encoded, offset)

	for i := 0; i < size; i++ {
		var kv [][]byte
		kv, offset = readSliceOfSlice(encoded, offset)
		key, err := utils.ToKey(kv[0])
		if err != nil {
			return PartyInfo{}, err
		}
		pi.recipients[*key] = string(kv[1])
	}

	var parties [][]byte
	parties, offset = readSliceOfSlice(encoded, offset)
	for _, party := range parties {
		pi.parties[string(party)] = true
	}

	return pi, nil
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

func readInt(src []byte, offset int) (int, int) {
	return int(binary.BigEndian.Uint64(src[offset:])), offset + 8
}

func writeSlice(src []byte, dest []byte, offset int) ([]byte, int) {
	length := len(src)
	dest, offset = writeInt(length, dest, offset)

	dest = confirmCapacity(dest, offset, length)
	copy(dest[offset:], src)
	return dest, offset + length
}

func readSliceToArray(src []byte, offset int, dest []byte) int {
	var length int
	length, offset = readInt(src, offset)
	offset += copy(dest, src[offset:offset + length])
	return offset
}

func readSlice(src []byte, offset int) ([]byte, int) {
	var length int
	length, offset = readInt(src, offset)
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
	arraySize, offset := readInt(src, offset)

	result := make([][]byte, arraySize)
	for i := 0; i < arraySize; i++ {
		var length int
		length, offset = readInt(src, offset)
		result[i] = append(
			result[i], src[offset:offset + length]...)
		offset += length
	}
	return result, offset
}